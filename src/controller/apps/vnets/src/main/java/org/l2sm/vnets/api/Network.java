package org.l2sm.vnets.api;

import java.util.ArrayList;
import java.util.List;

import org.onosproject.net.ConnectPoint;




public class Network {


    public String networkId;
    public List<ConnectPoint> networkEndpoints;
    public List<Long> tunnelIds;
    

    public Network() {

        this.networkEndpoints = new ArrayList<>();
        this.tunnelIds = new ArrayList<>();
    }

 
    public List<ConnectPoint> getNetworkEndpoints() {
        return networkEndpoints;
    }

    public String getNetworkId() {
        return networkId;
    }

    public List<Long> getIds() {
        return tunnelIds;
    }

    public Network clone(){
        Network newNetwork = new Network();
        newNetwork.networkId = networkId;
        newNetwork.networkEndpoints.addAll(networkEndpoints);
        newNetwork.tunnelIds.addAll(tunnelIds);
        return newNetwork;
    }

    public String toString(){
        StringBuffer buffer = new StringBuffer();
        buffer.append("Id: " + networkId + "\n");
        buffer.append("Endpoints: \n");
        for (ConnectPoint c: networkEndpoints){
            buffer.append(" -" + c + "\n"); 
        }
        buffer.append("Tunnel Ids: \n");
        for (Long l: tunnelIds){
            buffer.append(" -" + l + "\n"); 
        }
        return buffer.toString();
    }

}
