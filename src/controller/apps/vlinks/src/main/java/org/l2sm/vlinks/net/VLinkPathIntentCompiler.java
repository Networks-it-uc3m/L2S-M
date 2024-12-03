
package org.l2sm.vlinks.net;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

import org.onlab.util.KryoNamespace;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.DeviceId;
import org.onosproject.net.Link;
import org.onosproject.net.NetworkResource;
import org.onosproject.net.Path;
import org.onosproject.net.flow.DefaultTrafficSelector;
import org.onosproject.net.flow.DefaultTrafficTreatment;
import org.onosproject.net.flow.TrafficSelector;
import org.onosproject.net.flow.TrafficTreatment;
import org.onosproject.net.flowobjective.DefaultForwardingObjective;
import org.onosproject.net.flowobjective.ForwardingObjective;
import org.onosproject.net.flowobjective.Objective;
import org.onosproject.net.intent.FlowObjectiveIntent;
import org.onosproject.net.intent.Intent;
import org.onosproject.net.intent.IntentCompiler;
import org.onosproject.net.intent.IntentException;
import org.onosproject.net.intent.IntentExtensionService;
import org.onosproject.net.intent.TwoWayP2PIntent;
import org.onosproject.net.topology.PathService;
import org.onosproject.net.link.LinkService;
import org.onosproject.net.DefaultPath;
import org.onosproject.net.Path;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.Link;
import org.onosproject.net.DefaultLink;
import org.onosproject.net.ElementId;
import org.onlab.graph.ScalarWeight;
import org.onosproject.net.provider.ProviderId;
import org.osgi.service.component.annotations.Activate;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Deactivate;
import org.osgi.service.component.annotations.Reference;
import org.osgi.service.component.annotations.ReferenceCardinality;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


/**
 * An intent compiler for {@link org.l2sm.net.VLinkPathIntent}.
 */
@Component(immediate = true)
public class VLinkPathIntentCompiler implements IntentCompiler<VLinkPathIntent> {


        private final Logger log = LoggerFactory.getLogger(getClass());
        
        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected LinkService linkService;

        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected IntentExtensionService intentExtensionService;

        KryoNamespace namespace;

        @Activate
        public void activate() {
                intentExtensionService.registerCompiler(VLinkPathIntent.class, this);
        }

        @Deactivate
        public void deactivate() {
                intentExtensionService.unregisterCompiler(VLinkPathIntent.class);
        }

        @Override
        public List<Intent> compile(VLinkPathIntent intent, List<Intent> installable) {
                List<Intent> intentsToInstall = new ArrayList<>();
                List<Objective> toInstallObjectives = new ArrayList<>();
                List<DeviceId> toInstallDevices = new ArrayList<>();
                List<NetworkResource> resources = new ArrayList<>();


                if (intent.one().deviceId().equals(intent.two().deviceId())) {
                        return List.of(TwoWayP2PIntent.builder().appId(intent.appId()).one(intent.one())
                                        .two(intent.two()).priority(intent.priority()).build());
                }

                String[] path = intent.path();


                Iterable<Link> links = linkService.getActiveLinks();


                List<Link> path_links = new ArrayList<>();


                for (int i = 0; i < (path.length - 1); i++) {

                        for (Link link : links){
                                if (link.dst().elementId().toString().equals(path[i+1]) && link.src().elementId().toString().equals(path[i])){

                                        path_links.add(link);

                                }
                        }
                
                }


                resources.add(path_links.get(0));

                long tunnelId = intent.tunnelId();

                for (int i = 1; i < path_links.size(); i++) {

                        resources.add(path_links.get(i));
                        ConnectPoint port_1 = path_links.get(i - 1).dst();
                        ConnectPoint port_2 = path_links.get(i).src();
                        DeviceId deviceId = path_links.get(i).src().deviceId();
                        
                        toInstallObjectives.addAll(createInterFwdObjective(port_1, port_2, intent, tunnelId));
                        toInstallDevices.add(deviceId);
                        toInstallDevices.add(deviceId);

                }

                ConnectPoint one_tun_port = path_links.get(0).src();
                ConnectPoint two_tun_port = path_links.get(path_links.size() - 1).dst();

                // Objectives for port 1 (two objectives)
                toInstallObjectives.addAll(createEdgeFwdObjectives(intent.one(), one_tun_port, intent, tunnelId));
                toInstallDevices.add(intent.one().deviceId());
                toInstallDevices.add(intent.one().deviceId());

                // Objectives for port 2 (two objectives)
                toInstallObjectives.addAll(createEdgeFwdObjectives(intent.two(), two_tun_port, intent, tunnelId));
                toInstallDevices.add(intent.two().deviceId());
                toInstallDevices.add(intent.two().deviceId());

                FlowObjectiveIntent flowIntent = new FlowObjectiveIntent(intent.appId(), intent.key(), toInstallDevices,
                                toInstallObjectives, resources, intent.resourceGroup());

                intentsToInstall.add(flowIntent);

                return intentsToInstall;
        }

        private List<Objective> createEdgeFwdObjectives(ConnectPoint prov_port, ConnectPoint net_port,
                        VLinkPathIntent intent, long tunnelId) {

                TrafficSelector selector;
                TrafficTreatment treatment;
                List<Objective> objectives = new ArrayList<>(2);

                selector = DefaultTrafficSelector.builder().matchInPort(prov_port.port()).build();
                treatment = DefaultTrafficTreatment.builder().setTunnelId(tunnelId).setOutput(net_port.port())
                                .build();
                objectives.add(DefaultForwardingObjective.builder()
                                .withSelector(selector)
                                .withTreatment(treatment)
                                .fromApp(intent.appId())
                                .withPriority(intent.priority())
                                .withFlag(ForwardingObjective.Flag.SPECIFIC)
                                .add());

                selector = DefaultTrafficSelector.builder().matchInPort(net_port.port())
                                .matchTunnelId(tunnelId)
                                .build();
                treatment = DefaultTrafficTreatment.builder().setOutput(prov_port.port())
                                .build();
                objectives.add(DefaultForwardingObjective.builder()
                                .withSelector(selector)
                                .withTreatment(treatment)
                                .fromApp(intent.appId())
                                .withPriority(intent.priority())
                                .withFlag(ForwardingObjective.Flag.SPECIFIC)
                                .add());
                return objectives;
        }

        private List<Objective> createInterFwdObjective(ConnectPoint port1, ConnectPoint port2,
                        VLinkPathIntent intent, long tunnelId) {

                TrafficSelector selector;
                TrafficTreatment treatment;
                List<Objective> objectives = new ArrayList<>(2);

                selector = DefaultTrafficSelector.builder().matchInPort(port1.port()).matchTunnelId(tunnelId)
                                .build();
                treatment = DefaultTrafficTreatment.builder().setTunnelId(tunnelId).setOutput(port2.port())
                                .build();
                objectives.add(DefaultForwardingObjective.builder()
                                .withSelector(selector)
                                .withTreatment(treatment)
                                .fromApp(intent.appId())
                                .withPriority(intent.priority())
                                .withFlag(ForwardingObjective.Flag.SPECIFIC)
                                .add());

                selector = DefaultTrafficSelector.builder().matchInPort(port2.port()).matchTunnelId(tunnelId)
                                .build();
                treatment = DefaultTrafficTreatment.builder().setTunnelId(tunnelId).setOutput(port1.port())
                                .build();
                objectives.add(DefaultForwardingObjective.builder()
                                .withSelector(selector)
                                .withTreatment(treatment)
                                .fromApp(intent.appId())
                                .withPriority(intent.priority())
                                .withFlag(ForwardingObjective.Flag.SPECIFIC)
                                .add());
                return objectives;
        }

}

/*
 * List<Link> path_links = path.links();
 * // Intermediate neds rules installation
 * for (int i = 1; i < path_links.size(); i++) {
 * 
 * PortNumber port_1 = path_links.get(i - 1).dst().port();
 * PortNumber port_2 = path_links.get(i).src().port();
 * DeviceId deviceId = path_links.get(i).src().deviceId();
 * flowObjectiveService.forward(deviceId, createInterFwdObjective(port_1,
 * port_2, 13, 3));
 * flowObjectiveService.forward(deviceId, createInterFwdObjective(port_2,
 * port_1, 13, 3));
 * }
 * 
 * // Edge NEDs rules
 * PortNumber prov_port;
 * PortNumber net_port;
 * ForwardingObjective[] objectives;
 * 
 * Intent intent =
 * PointToPointIntent.builder().selector(null).treatment(null).build();
 * 
 * prov_port = net_cp1.port();
 * net_port = path_links.get(0).src().port();
 * objectives = createEdgeFwdObjectives(prov_port, net_port, 13, 3);
 * flowObjectiveService.forward(net_cp1.deviceId(), objectives[0]);
 * flowObjectiveService.forward(net_cp1.deviceId(), objectives[1]);
 * 
 * prov_port = net_cp2.port();
 * net_port = path_links.get(path_links.size() - 1).dst().port();
 * objectives = createEdgeFwdObjectives(prov_port, net_port, 13, 3);
 * flowObjectiveService.forward(net_cp2.deviceId(), objectives[0]);
 * flowObjectiveService.forward(net_cp2.deviceId(), objectives[1]);
 * 
 * private ForwardingObjective[] createEdgeFwdObjectives(PortNumber prov_port,
 * PortNumber net_port, long vni, int priority){
 * 
 * TrafficSelector selector;
 * TrafficTreatment treatment;
 * ForwardingObjective[] objectives = new ForwardingObjective[2];
 * 
 * selector = DefaultTrafficSelector.builder().matchInPort(prov_port).build();
 * treatment =
 * DefaultTrafficTreatment.builder().setTunnelId(vni).setOutput(net_port)
 * .build();
 * objectives[0] = DefaultForwardingObjective.builder()
 * .withSelector(selector)
 * .withTreatment(treatment)
 * .fromApp(this.app_id)
 * .withPriority(priority)
 * .withFlag(ForwardingObjective.Flag.SPECIFIC)
 * .add();
 * 
 * selector =
 * DefaultTrafficSelector.builder().matchInPort(net_port).matchTunnelId(vni).
 * build();
 * treatment = DefaultTrafficTreatment.builder().setOutput(prov_port)
 * .build();
 * objectives[1] = DefaultForwardingObjective.builder()
 * .withSelector(selector)
 * .withTreatment(treatment)
 * .fromApp(this.app_id)
 * .withPriority(priority)
 * .withFlag(ForwardingObjective.Flag.SPECIFIC)
 * .add();
 * return objectives;
 * }
 * 
 * private ForwardingObjective createInterFwdObjective(PortNumber port1,
 * PortNumber port2, long vni, int priority){
 * 
 * TrafficSelector selector;
 * TrafficTreatment treatment;
 * 
 * selector =
 * DefaultTrafficSelector.builder().matchInPort(port1).matchTunnelId(vni).build(
 * );
 * treatment =
 * DefaultTrafficTreatment.builder().setTunnelId(vni).setOutput(port2).build();
 * return DefaultForwardingObjective.builder()
 * .withSelector(selector)
 * .withTreatment(treatment)
 * .fromApp(this.app_id)
 * .withPriority(priority)
 * .withFlag(ForwardingObjective.Flag.SPECIFIC)
 * .add();
 * 
 * }
 * 
 * // Objectives for intermediate nodes
 * /*
 * TrafficSelector selector =
 * DefaultTrafficSelector.builder().matchTunnelId(tunnelId).build();
 * TrafficTreatment treatment =
 * DefaultTrafficTreatment.builder().setTunnelId(tunnelId).build();
 * 
 * TwoWayP2PIntent interIntent = TwoWayP2PIntent.builder()
 * .appId(intent.appId())
 * .one(one_tun_port)
 * .two(two_tun_port)
 * .selector(selector)
 * .treatment(treatment)
 * .priority(intent.priority())
 * .build();
 */

/*
 * LinkCollectionIntent linkIntent = LinkCollectionIntent.builder()
 * .appId(intent.appId())
 * .applyTreatmentOnEgress(false)
 * .filteredEgressPoints(Set.of(new FilteredConnectPoint(intent.one())))
 * .filteredIngressPoints(Set.of(new FilteredConnectPoint(intent.two())))
 * .key(intent.key())
 * .selector(selector)
 * .links(new HashSet<>(path_links))
 * .treatment(treatment)
 * .build();
 */