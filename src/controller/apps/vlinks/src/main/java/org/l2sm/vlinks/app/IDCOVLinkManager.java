package org.l2sm.vlinks.app;

import static org.onlab.util.Tools.groupedThreads;

import java.security.SecureRandom;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Objects;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

import org.apache.karaf.shell.api.action.lifecycle.Service;
import org.l2sm.vlinks.api.IDCOVLinkService;
import org.l2sm.vlinks.api.IDCOVLinkServiceException;
import org.l2sm.vlinks.api.VLinkNetwork;
import org.l2sm.vlinks.net.VLinkPathIntent;
import org.onlab.packet.Ethernet;
import org.onlab.packet.MacAddress;
import org.onosproject.core.ApplicationId;
import org.onosproject.core.CoreService;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.Path;
import org.onosproject.net.DefaultPath;
import org.onosproject.net.config.NetworkConfigService;
import org.onosproject.net.device.DeviceService;
import org.onosproject.net.flow.DefaultFlowRule;
import org.onosproject.net.flow.DefaultTrafficSelector;
import org.onosproject.net.flow.DefaultTrafficTreatment;
import org.onosproject.net.flow.FlowRule;
import org.onosproject.net.flow.TrafficSelector;
import org.onosproject.net.flow.TrafficTreatment;
import org.onosproject.net.group.GroupService;
import org.onosproject.net.intent.FlowRuleIntent;
import org.onosproject.net.intent.Intent;
import org.onosproject.net.intent.IntentEvent;
import org.onosproject.net.intent.IntentListener;
import org.onosproject.net.intent.IntentService;
import org.onosproject.net.intent.Key;
import org.onosproject.net.intent.ObjectiveTrackerService;
import org.onosproject.net.intent.PathIntent;
import org.onosproject.net.packet.DefaultOutboundPacket;
import org.onosproject.net.packet.OutboundPacket;
import org.onosproject.net.packet.PacketContext;
import org.onosproject.net.packet.PacketPriority;
import org.onosproject.net.packet.PacketProcessor;
import org.onosproject.net.packet.PacketService;
import org.osgi.service.component.annotations.Activate;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Deactivate;
import org.osgi.service.component.annotations.Reference;
import org.osgi.service.component.annotations.ReferenceCardinality;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.primitives.Longs;

@Component(immediate = true)
@Service
public class IDCOVLinkManager implements IDCOVLinkService {

    private static final int VIRTUAL_LINK_PRIORITY = PacketPriority.HIGH3.priorityValue();
    private static final PacketPriority ARP_TO_CONTROLLER_PRIORITY = PacketPriority.HIGH2;
    private static final int VIRTUAL_NETWORK_CORE_PRIORITY = PacketPriority.HIGH1.priorityValue();
    private static final int VIRTUAL_NETWORK_EDGE_PRIORITY = PacketPriority.HIGH1.priorityValue();

    private final Logger log = LoggerFactory.getLogger(getClass());

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected CoreService coreService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected PacketService packetService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected IntentService intentService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected DeviceService deviceService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected NetworkConfigService networkConfigService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected GroupService groupService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected ObjectiveTrackerService objectiveTrackerService;

    private IDCOVLinkDatabase database;
    private TunnelIdProvider tunnelIdProvider;

    private ArpProxyPacketProcessor packetProcessor;
    private VNFLocationProvider vnfLocationProvider;
    private CustomIntentListener intentListener;

    private ExecutorService genericEventHandler;

    private ApplicationId appId;

    @Activate
    protected void activate() {
        log.info("Starting IDCO");
        appId = coreService.registerApplication("org.l2sm.app");

        this.database = new IDCOVLinkDatabase(log);

        packetProcessor = new ArpProxyPacketProcessor();
        packetService.addProcessor(packetProcessor, PacketProcessor.director(2));

        vnfLocationProvider = new VNFLocationProvider();
        packetService.addProcessor(vnfLocationProvider, PacketProcessor.advisor(1));

        intentListener = new CustomIntentListener();
        intentService.addListener(intentListener);

        genericEventHandler = Executors.newFixedThreadPool(4, groupedThreads("idco/event-handler", "worker-%d", log));

        vnfLocationProvider.requestIntercepts();

        tunnelIdProvider = new TunnelIdProvider();

        log.info("IDCO was started");

    }

    @Deactivate
    protected void deactivate() {
        log.info("Starting the IDCO cleaning process");

        log.info("Removing interceptors and packet processors");
        vnfLocationProvider.withdrawIntercepts();
        packetService.removeProcessor(packetProcessor);
        packetService.removeProcessor(vnfLocationProvider);

        log.info("Shutting down the event handler");
        genericEventHandler.shutdown();
        try {
            genericEventHandler.awaitTermination(60, TimeUnit.SECONDS);
        } catch (InterruptedException e) {
            log.error("Could not shutdown thread executors correctly");
        }

        log.info("Withdrawing all the intents");
        database.getAllIntents().forEach(intentKey -> {
            Intent intent = intentService.getIntent(intentKey);
            if (intentKey != null)
                intentService.withdraw(intent);
        });

        log.info("Clearing database");
        database.cleanVLinkDatabases();
        intentService.removeListener(intentListener);

        log.info("IDCO has stopped");
    }

    public void createVLinkNetwork(String networkVlinkId, ConnectPoint networkVlinkFromEndpoint, ConnectPoint networkVlinkToEndpoint, String[] vLinkPath) throws IDCOVLinkServiceException {
        
        genericEventHandler.submit(() -> {
            log.info("Creating network: " + networkVlinkId);
            log.info("Adding new Path from " + networkVlinkFromEndpoint.toString() + " to " + networkVlinkToEndpoint.toString());
            database.lockVLinkNetwork(networkVlinkId);


            log.info("Registering new network");
            database.registerVLinkNetwork(networkVlinkId);
            Long tunnelId = tunnelIdProvider.getNewId(); 
            
            log.info("Adding port " + networkVlinkFromEndpoint.toString() + " to network " + networkVlinkId + " to the database");
            database.addPortToVLinkNetwork(networkVlinkId, networkVlinkFromEndpoint, tunnelId);
            log.info("Adding port " + networkVlinkToEndpoint.toString() + " to network " + networkVlinkId + " to the database");
            database.addPortToVLinkNetwork(networkVlinkId, networkVlinkToEndpoint, tunnelId);
            log.info("Ports added to the database");

            VLinkNetwork network = database.getVLinkNetwork(networkVlinkId);
            long[] ids = Longs.toArray(network.getIds());

            Intent intent = null;
            Key intentKey = Key.of("idco-main-" + networkVlinkId, appId);
            log.info("Creating main intent for network " + networkVlinkId);
            intent = VLinkPathIntent.builder()
                    .key(intentKey)  
                    .appId(appId)
                    .one(networkVlinkFromEndpoint)
                    .two(networkVlinkToEndpoint)
                    .path(vLinkPath)
                    .priority(VIRTUAL_LINK_PRIORITY)
                    .tunnelID(tunnelId)
                    .build();

            log.info("Submitting new main intent for network " + networkVlinkId);
            intentService.submit(intent);
            log.info("Adding main intent to database for the network " + networkVlinkId);
            database.addMainIntent(networkVlinkId, intentKey);
            database.unlockVLinkNetwork(networkVlinkId);
            log.info("The network " + networkVlinkId + " from " + networkVlinkFromEndpoint + " to " + networkVlinkToEndpoint + " was correctly created");
        });
    }

    public void deleteVLinkNetwork(String networkVlinkId) throws IDCOVLinkServiceException {
        genericEventHandler.submit(() -> {
            log.info("Deleting network " + networkVlinkId);
            database.lockVLinkNetwork(networkVlinkId);
            Collection<Key> net_intent = database.getVLinkNetworkIntents(networkVlinkId);

            log.info("Deleting intents for network " + networkVlinkId);
            net_intent.forEach(intentKey -> {
                Intent intent = intentService.getIntent(intentKey);
                if (intent != null)
                    intentService.withdraw(intent);
            });

            log.info("Deleting network "+ networkVlinkId + "from the database");
            database.deleteVLinkNetwork(networkVlinkId);

            log.info("The network with id \"" + networkVlinkId + "\" has been deleted");
            database.unlockVLinkNetwork(networkVlinkId);
        });

    }


    public VLinkNetwork getVLinkNetwork(String networkVlinkId) throws IDCOVLinkServiceException {

        Future<VLinkNetwork> future = genericEventHandler.submit(() -> {
            log.info("Retrieving network " + networkVlinkId);
            database.lockVLinkNetwork(networkVlinkId);

            VLinkNetwork network = database.getVLinkNetwork(networkVlinkId);
            database.unlockVLinkNetwork(networkVlinkId);
            return network;
        });

        try {
            return future.get();
        } catch (Exception e) {
            return null;
        }

    }

    class ArpProxyPacketProcessor implements PacketProcessor {

        public ArpProxyPacketProcessor() {

        }

        @Override
        public void process(PacketContext context) {

            // Verify valid context
            if (context == null || context.isHandled()) {
                return;
            }
            // Verify valid Ethernet packet
            Ethernet eth = context.inPacket().parsed();
            if (eth == null) {
                return;
            }

            genericEventHandler.submit(() -> processPacketInternal(context));
        }

        public void processPacketInternal(PacketContext context) {

            Ethernet eth = context.inPacket().parsed();

            MacAddress dstMac = eth.getDestinationMAC();
            ConnectPoint heardPort = context.inPacket().receivedFrom();

            String mscsId = database.getVLinkNetworkIdForPort(heardPort);
            if (mscsId == null) {
                return;
            }

            database.lockVLinkNetwork(mscsId);

            if (!(dstMac.isBroadcast() || dstMac.isMulticast())) {
                ConnectPoint hostLocation = database.getHostLocation(mscsId, dstMac);
                if (hostLocation != null) {
                    TrafficTreatment treatment = DefaultTrafficTreatment.builder().setOutput(hostLocation.port())
                            .build();
                    OutboundPacket outboundPacket = new DefaultOutboundPacket(hostLocation.deviceId(), treatment,
                            context.inPacket().unparsed());
                    packetService.emit(outboundPacket);
                    context.block();
                    database.unlockVLinkNetwork(mscsId);
                    return;
                }

            }

            Collection<ConnectPoint> connectPoints = database.getPortsOfVLinkNetworkGivenPort(heardPort);

            connectPoints.forEach(point -> {
                TrafficTreatment treatment = DefaultTrafficTreatment.builder().setOutput(point.port()).build();
                OutboundPacket outboundPacket = new DefaultOutboundPacket(point.deviceId(), treatment,
                        context.inPacket().unparsed());
                packetService.emit(outboundPacket);
            });

            context.block();
            log.info("Proxying packet for: " + dstMac.toString() + " in network " + mscsId.toString());
            database.unlockVLinkNetwork(mscsId);
        }

    }

    private class VNFLocationProvider implements PacketProcessor {

        /**
         * Request packet intercepts.
         */
        private void requestIntercepts() {
            // Use ARP
            TrafficSelector.Builder selector = DefaultTrafficSelector.builder()
                    .matchEthType(Ethernet.TYPE_ARP);
            packetService.requestPackets(selector.build(), ARP_TO_CONTROLLER_PRIORITY, appId);

        }

        /**
         * Withdraw packet intercepts.
         */
        private void withdrawIntercepts() {
            TrafficSelector.Builder selector = DefaultTrafficSelector.builder();
            selector.matchEthType(Ethernet.TYPE_ARP);
            packetService.cancelPackets(selector.build(), ARP_TO_CONTROLLER_PRIORITY, appId);
        }

        @Override
        public void process(PacketContext context) {
            // Verify valid context
            if (context == null) {
                return;
            }
            // Verify valid Ethernet packet
            Ethernet eth = context.inPacket().parsed();
            if (eth == null) {
                return;
            }
            MacAddress srcMac = eth.getSourceMAC();
            if (srcMac.isBroadcast() || srcMac.isMulticast()) {
                return;
            }

            genericEventHandler.submit(() -> processPacketInternal(context), Objects.hash(srcMac));
        }

        private void processPacketInternal(PacketContext context) {
            Ethernet eth = context.inPacket().parsed();

            ConnectPoint heardOn = context.inPacket().receivedFrom();

            // If this arrived on control port, bail out.
            if (heardOn.port().isLogical()) {
                return;
            }

            MacAddress hostId = eth.getSourceMAC();

            if (eth.getEtherType() == Ethernet.TYPE_ARP) {
                detectedHost(hostId, heardOn, context);
            }
        }

        public void detectedHost(MacAddress macAddress, ConnectPoint hostLocation, PacketContext context) {

            String mscsId = database.getVLinkNetworkIdForPort(hostLocation);
            if (mscsId == null) {
                return;
            }
            database.lockVLinkNetwork(mscsId);
            log.info("New packet received: " + macAddress.toString() + " for network " + mscsId.toString());

            ConnectPoint lastLocation = database.getHostLocation(mscsId, macAddress);

            if (lastLocation != null) {
                if (!lastLocation.equals(hostLocation)) {
                    log.warn("The host " + macAddress.toString() + " in network " + mscsId
                            + " has changed its location. The system does not supporthost mobility");
                }
                database.unlockVLinkNetwork(mscsId);
                return;
            }

            Long tunnelId = database.getTunnelIdOfPort(hostLocation);
            if (tunnelId == null) {
                context.block();
                database.unlockVLinkNetwork(mscsId);
                return;
            }

            Collection<ConnectPoint> connectPoint = database.getPortsOfVLinkNetworkGivenPort(hostLocation);

            List<FlowRule> rules = connectPoint.stream()
                    .map(point -> createRule(macAddress, hostLocation, point, tunnelId))
                    .collect(Collectors.toList());

            Key key = generateHostIntentKey(macAddress, mscsId);

            FlowRuleIntent ruleIntent = new FlowRuleIntent(appId, key, rules,
                    Collections.emptyList(), PathIntent.ProtectionType.PRIMARY, null);

            intentService.submit(ruleIntent);
            database.addIntentToVLinkNetwork(mscsId, key);
            database.setHostLocation(mscsId, macAddress, hostLocation);
            database.unlockVLinkNetwork(mscsId);
        }

        private FlowRule createRule(MacAddress address, ConnectPoint cp, ConnectPoint otherCp, long tunnelId) {
            TrafficTreatment treatment = DefaultTrafficTreatment.builder().setTunnelId(tunnelId).transition(1).build();
            TrafficSelector selector = DefaultTrafficSelector.builder().matchEthDst(address).matchInPort(otherCp.port())
                    .build();
            return DefaultFlowRule.builder().fromApp(appId)
                    .withPriority(VIRTUAL_NETWORK_EDGE_PRIORITY).withTreatment(treatment)
                    .withSelector(selector).makePermanent().forDevice(otherCp.deviceId()).build();

        }

    }

    class CustomIntentListener implements IntentListener {

        @Override
        public void event(IntentEvent event) {
            genericEventHandler.submit(() -> handleEvent(event));
        }

        public void handleEvent(IntentEvent event) {
            Intent intent = event.subject();
            log.info("Intent event: " + event.type().name());
            switch (event.type()) {
                case FAILED:
                    // objectiveTrackerService.addTrackedResources(intent.key(), );
                    break;
                case INSTALLED:
                    break;
                case WITHDRAWN:
                    intentService.purge(intent);
                    break;
                default:
                    break;
            }

        }
    }

    /************ UTILS ***************************************/

    private Key generateHostIntentKey(MacAddress hostMac, String mscsId) {

        return Key.of("idco-host-" + mscsId.toString() + "-" + hostMac.toString(), appId);
    }

    /*
     * We are not focusing on security in this implementation. Future
     * implementations
     * will include a more secure Tunnel Id provider.
     * TODO: check valid vxlan tunnels
     */
    static class TunnelIdProvider {

        /*
         * Linear Congruent generator for 24 bit numbers
         * The parameters c and a are chosen to make the period 2*24:
         * - c is relatively prime to 2^24
         * - 2 is a factor of a - 1
         * - 4 is a factor of a - 1
         * 
         * TODO: generate this values dinamically
         */
        private long c = 16777213;
        private long a = 258088 + 1;
        private long modulus = 16777216;
        private long last_id;
        private long count;

        public TunnelIdProvider() {
            IdGenerator generator = new IdGenerator();
            this.last_id = generator.nextId();
            count = 0;
        }

        public Long getNewId() {
            

            long id;
            if (count == modulus) {
                return null;
            }
            id = last_id;
            this.last_id = (last_id * a + c) % modulus;
            count++;

            return id;
        }

        private class IdGenerator extends SecureRandom {
            public IdGenerator() {
                super();
            }

            public long nextId() {
                return next(24);
            }
        }
    }

}
