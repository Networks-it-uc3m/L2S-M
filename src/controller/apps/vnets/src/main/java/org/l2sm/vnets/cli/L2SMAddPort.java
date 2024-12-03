package org.l2sm.vnets.cli;

import org.apache.karaf.shell.api.action.Argument;
import org.apache.karaf.shell.api.action.Command;
import org.apache.karaf.shell.api.action.lifecycle.Service;
import org.l2sm.vnets.api.IDCOService;
import org.onosproject.cli.AbstractShellCommand;
import org.onosproject.net.ConnectPoint;

@Service
@Command(scope = "onos", name = "l2sm-add-port", description = "Add a port to an existing network")

public class L2SMAddPort extends AbstractShellCommand {

    @Argument(index = 0, name = "networkId", description = "networkId", required = true, multiValued = false)
    String networkId = null;

    @Argument(index = 1, name = "networkEndpoint", description = "networkEndpoint", required = true, multiValued = false)
    String networkEndpoint = null;

    @Override
    protected void doExecute() {
        IDCOService idcoService = get(IDCOService.class);
        try {
            idcoService.addPort(networkId, ConnectPoint.deviceConnectPoint(networkEndpoint));
        } catch (Exception e) {
            print("Error ocurred");
            print(e.toString());
        }
    }

}
