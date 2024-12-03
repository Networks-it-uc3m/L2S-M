
package org.l2sm.vnets.net;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collection;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import org.onosproject.core.ApplicationId;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.DeviceId;
import org.onosproject.net.Link;
import org.onosproject.net.NetworkResource;
import org.onosproject.net.Path;
import org.onosproject.net.PortNumber;
import org.onosproject.net.flow.DefaultFlowRule;
import org.onosproject.net.flow.DefaultTrafficSelector;
import org.onosproject.net.flow.DefaultTrafficTreatment;
import org.onosproject.net.flow.FlowRule;
import org.onosproject.net.flow.TrafficSelector;
import org.onosproject.net.flow.TrafficTreatment;
import org.onosproject.net.flowobjective.DefaultForwardingObjective;
import org.onosproject.net.flowobjective.DefaultNextObjective;
import org.onosproject.net.flowobjective.DefaultNextTreatment;
import org.onosproject.net.flowobjective.FlowObjectiveService;
import org.onosproject.net.flowobjective.ForwardingObjective;
import org.onosproject.net.flowobjective.ForwardingObjective.Flag;
import org.onosproject.net.flowobjective.NextObjective;
import org.onosproject.net.flowobjective.NextTreatment;
import org.onosproject.net.flowobjective.Objective;
import org.onosproject.net.group.GroupService;
import org.onosproject.net.intent.FlowObjectiveIntent;
import org.onosproject.net.intent.FlowRuleIntent;
import org.onosproject.net.intent.Intent;
import org.onosproject.net.intent.IntentCompiler;
import org.onosproject.net.intent.IntentException;
import org.onosproject.net.intent.IntentExtensionService;
import org.onosproject.net.intent.PathIntent;
import org.onosproject.net.topology.PathService;
import org.osgi.service.component.annotations.Activate;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Deactivate;
import org.osgi.service.component.annotations.Reference;
import org.osgi.service.component.annotations.ReferenceCardinality;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * An intent compiler for {@link org.l2sm.vnets.net.VirtualLinkIntent}.
 */
@Component(immediate = true)
public class VirtualNetworkIntentCompiler implements IntentCompiler<VirtualNetworkIntent> {

        private final static int TABLE_ONE = 1;

        private final Logger log = LoggerFactory.getLogger(getClass());

        private ApplicationId appId;

        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected PathService pathService;

        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected IntentExtensionService intentExtensionService;

        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected FlowObjectiveService flowObjectiveService;

        @Reference(cardinality = ReferenceCardinality.MANDATORY)
        protected GroupService groupService;

        @Activate
        public void activate() {
                intentExtensionService.registerCompiler(VirtualNetworkIntent.class, this);
        }

        @Deactivate
        public void deactivate() {
                intentExtensionService.unregisterCompiler(VirtualNetworkIntent.class);
        }

        @Override
        public List<Intent> compile(VirtualNetworkIntent intent, List<Intent> installable) {
                log.info("Compiling virtual network intent");
                appId = intent.appId();
                List<Intent> intentsToInstall = new ArrayList<>();

                for (int i = 0; i < intent.connectPoints.length; i++) {
                        ConnectPoint rootPoint = intent.connectPoints[i];
                        HashSet<NetworkResource> resources = new HashSet<>();
                        PointToMultipointNode rootNode = getPointToMultipointTree(rootPoint, intent.connectPoints,
                                        resources);
                        log.info(rootNode.toString());
                        intentsToInstall.addAll(generateTreeIntent(rootNode, intent, (intent.tunnelIds())[i], resources));
                }

                return intentsToInstall;
        }

        private PointToMultipointNode getPointToMultipointTree(ConnectPoint rootPoint,
                        ConnectPoint[] completePointsList, Set<NetworkResource> resources) {

                PointToMultipointNode rootNode = new PointToMultipointNode(rootPoint);
                for (ConnectPoint dp : completePointsList) {
                        if (dp.equals(rootPoint)) {
                                continue;
                        }

                        List<ConnectPoint> points = new ArrayList<>();
                        if (!rootPoint.deviceId().equals(dp.deviceId())) {
                                List<Link> pathLinks = calculatePathLinks(rootPoint.deviceId(), dp.deviceId());
                                if (pathLinks == null) {
                                        return null;
                                }
                                for (Link link : pathLinks) {
                                        points.add(link.src());
                                        points.add(link.dst());
                                }
                                resources.addAll(pathLinks);
                        }

                        log.info("Añadiendo path al arbol");

                        PointToMultipointNode currentNode = rootNode;

                        // It will always be an even size
                        assert (points.size() % 2 == 0);
                        for (int i = 0; i < points.size(); i += 2) {
                                ConnectPoint port = points.get(i);
                                ConnectPoint nextPort = points.get(i + 1);
                                PointToMultipointNode nextNode = null;
                                if ((nextNode = currentNode.getChild(port)) == null) {
                                        nextNode = currentNode.addChild(port, nextPort);
                                }
                                currentNode = nextNode;
                        }
                        currentNode.addChild(dp, null);
                }

                return rootNode;
        }

        private List<Link> calculatePathLinks(DeviceId id1, DeviceId id2) {
                Set<Path> paths = pathService.getPaths(id1, id2);
                Path path = paths.iterator().hasNext() ? paths.iterator().next() : null;
                if (path == null) {
                        throw new IntentException("El path es null");
                }
                return path.links();
        }

        private Collection<Intent> generateTreeIntent(PointToMultipointNode rootNode, VirtualNetworkIntent intent, long tunnelId,
                        Collection<NetworkResource> resources) {
                List<FlowRule> toInstallRules = new ArrayList<>();
                List<Objective> flowObjectives = new ArrayList<>();
                List<DeviceId> devices = new ArrayList<>();

                generateNodeRules(rootNode, true, toInstallRules, flowObjectives, devices, tunnelId, intent);

                /*
                 * We mix rules and objectives to avoid having to change other parts of the code.
                 * Everything should be objectives in the future
                 * True reason: soy una tía chulísima y los mezclo porque quiero.
                 */
                FlowRuleIntent ruleIntent = new FlowRuleIntent(intent.appId(), intent.key(), toInstallRules,
                                resources, PathIntent.ProtectionType.PRIMARY, intent.resourceGroup());
                FlowObjectiveIntent flowIntent = new FlowObjectiveIntent(appId, intent.key(), devices, flowObjectives,
                                resources, null);

                return Arrays.asList(flowIntent,ruleIntent);
        }

        private void generateNodeRules(PointToMultipointNode node, boolean isFirst, List<FlowRule> rules,
                        List<Objective> flowObjectives, List<DeviceId> devices, long tunnelId,
                        VirtualNetworkIntent intent) {

                DeviceId deviceId = node.getRootPort().deviceId();
                PortNumber rootPort = node.getRootPort().port();

                TrafficTreatment convergencePortTreatment = portTreatment(rootPort, isFirst ? null : tunnelId);

                // Rules for convergence path
                node.getChildren().forEach((designatedPort, nextNode) -> {

                        boolean isFinalPort = nextNode == null;

                        if (isFinalPort) {
                                rules.add(DefaultFlowRule.builder().forDevice(deviceId).fromApp(appId)
                                                .withPriority(intent.priority())
                                                .withSelector(portSelector(designatedPort.port(), tunnelId))
                                                .withTreatment(convergencePortTreatment).makePermanent()
                                                .forTable(TABLE_ONE).forDevice(deviceId).build());
                        } else {
                                rules.add(DefaultFlowRule.builder().forDevice(deviceId).fromApp(appId)
                                                .withPriority(intent.priority())
                                                .withSelector(portSelector(designatedPort.port(), tunnelId))
                                                .withTreatment(convergencePortTreatment).makePermanent()
                                                .forDevice(deviceId).build());
                        }

                });

               

                // Rules for divervence path
                int nextId = flowObjectiveService.allocateNextId();
                NextObjective.Builder nextObjectiveBuilder = DefaultNextObjective.builder()
                .withPriority(intent.priority() - 5).withType(NextObjective.Type.BROADCAST).fromApp(appId)
                .withId(nextId);

                node.getChildren().entrySet().stream().forEach((e) -> {
                        ConnectPoint port = e.getKey();
                        PointToMultipointNode nextNode = e.getValue();
                        NextTreatment nextTreatment = DefaultNextTreatment
                                        .of(portTreatment(port.port(), nextNode == null ? null : tunnelId));
                        nextObjectiveBuilder.addTreatment(nextTreatment);
                });

                ForwardingObjective.Builder builder = DefaultForwardingObjective.builder().fromApp(appId)
                                .makePermanent().withFlag(Flag.SPECIFIC).nextStep(nextId);

                if (isFirst) {

                        // Broadcast and multicast traffic
                        builder.withSelector(DefaultTrafficSelector.builder().matchInPort(rootPort)
                                        .build()).withPriority(intent.priority() - 5);

                        /*
                         * builder.withSelector(DefaultTrafficSelector.builder().matchInPort(rootPort)
                         * .build()).withPriority(NO_MATCH_PRIORITY);
                         */
                } else {
                        builder.withSelector(DefaultTrafficSelector.builder().matchTunnelId(tunnelId)
                                        .matchInPort(rootPort).build()).withPriority(intent.priority());
                }

                flowObjectives.add(nextObjectiveBuilder.add());
                flowObjectives.add(builder.add());
                devices.add(deviceId);
                devices.add(deviceId);

                // Propagate to the next nodes
                node.getChildrenNodes().forEach(child -> {
                        if (child != null) {
                                generateNodeRules(child, false, rules, flowObjectives, devices, tunnelId, intent);
                        }
                });

        }

        private static TrafficSelector portSelector(PortNumber port, Long vni) {
                if (vni == null) {
                        return DefaultTrafficSelector.builder().matchInPort(port).build();
                }
                return DefaultTrafficSelector.builder().matchInPort(port).matchTunnelId(vni).build();
        }

        private static TrafficTreatment portTreatment(PortNumber port, Long vni) {
                if (vni == null) {
                        return DefaultTrafficTreatment.builder().setOutput(port).build();
                }
                return DefaultTrafficTreatment.builder().setTunnelId(vni).setOutput(port).build();
        }

}
