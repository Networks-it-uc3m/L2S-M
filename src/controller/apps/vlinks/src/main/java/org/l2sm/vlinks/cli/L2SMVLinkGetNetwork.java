package org.l2sm.vlinks.cli;

import org.apache.karaf.shell.api.action.Argument;
import org.apache.karaf.shell.api.action.Command;
import org.apache.karaf.shell.api.action.lifecycle.Service;
import org.l2sm.vlinks.api.IDCOVLinkService;
import org.l2sm.vlinks.api.VLinkNetwork;
import org.onosproject.cli.AbstractShellCommand;

@Service
@Command(scope = "onos", name = "l2sm-vlink-get-network", description = "Create a VLink network")


public class L2SMVLinkGetNetwork extends AbstractShellCommand {

    @Argument(index = 0, name = "networkVlinkId", description = "networkVlinkId", required = true, multiValued = false)
    String networkVlinkId = null;

    @Override
    protected void doExecute() {
        print("Getting network...");
        IDCOVLinkService idcoVlinkService = get(IDCOVLinkService.class);
        try {
           VLinkNetwork networkVlink = idcoVlinkService.getVLinkNetwork(networkVlinkId);
            if (networkVlink == null){
                print("Network '"+networkVlinkId+"' does not exist");
                return;
            }
           print(networkVlink.toString());
        } catch (Exception e) {
            print("Error ocurred");
            print(e.toString());
        }
    }

}
