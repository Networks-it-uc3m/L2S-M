package org.l2sm.vlinks.cli;

import org.apache.karaf.shell.api.action.Argument;
import org.apache.karaf.shell.api.action.Command;
import org.apache.karaf.shell.api.action.lifecycle.Service;
import org.l2sm.vlinks.api.IDCOVLinkService;
import org.onosproject.cli.AbstractShellCommand;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.Link;
import org.onosproject.net.DefaultLink;
import org.onlab.graph.ScalarWeight;
import org.onosproject.net.DefaultPath;
import org.onosproject.net.Path;
import org.onosproject.net.provider.ProviderId;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;


@Service
@Command(scope = "onos", name = "l2sm-vlink-create-network", description = "Creates the new Network with the determined Path")

public class L2SMVLinkCreateNetwork extends AbstractShellCommand {

    @Argument(index = 0, name = "networkVlinkId", description = "networkVlinkId", required = true, multiValued = false)
    String networkVlinkId = null;

    @Argument(index = 1, name = "networkVlinkFromEndpoint", description = "networkVlinkFromEndpoint", required = true, multiValued = false)
    String networkVlinkFromEndpoint = null;

    @Argument(index = 2, name = "networkVlinkToEndpoint", description = "networkVlinkToEndpoint", required = true, multiValued = false)
    String networkVlinkToEndpoint = null;

    @Argument(index = 3, name = "vLinkPath", description = "vLinkPath", required = true, multiValued = false)
    String[] vLinkPath;

    @Override
    protected void doExecute() {
        print("Creating a new Network...");
        IDCOVLinkService idcoVlinkService = get(IDCOVLinkService.class);

        try {

            idcoVlinkService.createVLinkNetwork(networkVlinkId, ConnectPoint.deviceConnectPoint(networkVlinkFromEndpoint),ConnectPoint.deviceConnectPoint(networkVlinkToEndpoint), vLinkPath); 
            print("Success! Network:" + networkVlinkId + " has been created");
            print(networkVlinkId + " From EndPoint: " + networkVlinkFromEndpoint);
            print(networkVlinkId + " To EndPoint: " + networkVlinkToEndpoint);
            print(networkVlinkId + " Path: ");
            for (int i= 0; i < vLinkPath.length; i++){
                print("Node " + i + " : " + vLinkPath[i]);
            }

        } catch (Exception e) {
            print("Error ocurred");
            print(e.toString());
        }
    }

}
