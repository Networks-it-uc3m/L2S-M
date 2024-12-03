package org.l2sm.vnets.cli;

import org.apache.karaf.shell.api.action.Argument;
import org.apache.karaf.shell.api.action.Command;
import org.apache.karaf.shell.api.action.lifecycle.Service;
import org.l2sm.vnets.api.IDCOService;
import org.onosproject.cli.AbstractShellCommand;

@Service
@Command(scope = "onos", name = "l2sm-create-network", description = "Create a network")


public class L2SMCreateNetwork extends AbstractShellCommand {

    @Argument(index = 0, name = "networkId", description = "networkId", required = true, multiValued = false)
    String networkId = null;

    @Override
    protected void doExecute() {
        IDCOService idcoService = get(IDCOService.class);
        try {
            idcoService.createVirtualNetwork(networkId);
        } catch (Exception e) {
            print("Error ocurred");
            print(e.toString());
        }
    }

}
