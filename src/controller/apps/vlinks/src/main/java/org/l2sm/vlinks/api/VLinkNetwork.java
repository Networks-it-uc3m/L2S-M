package org.l2sm.vlinks.api;

import java.util.ArrayList;
import java.util.List;

import org.onosproject.net.ConnectPoint;




public class VLinkNetwork {

    
    // This class has 3 parameters:
    //  - networkVlinkId:          It is the name of the Link (String)
    //  - networkVlinkEndpoints:   They are the From and To Points (List<ConnectPoint>)
    //  - tunnelIds:               Still trying to figure thisone out (List<Long>)
    
    // This class has the following commands:
    //  - get
    //  - clone
    //  - to String
    


    public String networkVlinkId;
    public List<ConnectPoint> networkVlinkEndpoints;
    public List<Long> tunnelIds;
    

    public VLinkNetwork() {

        this.networkVlinkEndpoints = new ArrayList<>();
        this.tunnelIds = new ArrayList<>();
    }

 
    public List<ConnectPoint> getVLinkNetworkEndpoints() {
        return networkVlinkEndpoints;
    }

    public String getVLinkNetworkId() {
        return networkVlinkId;
    }

    public List<Long> getIds() {
        return tunnelIds;
    }

    public VLinkNetwork clone(){
        VLinkNetwork newVLinkNetwork = new VLinkNetwork();
        newVLinkNetwork.networkVlinkId = networkVlinkId;
        newVLinkNetwork.networkVlinkEndpoints.addAll(networkVlinkEndpoints);
        newVLinkNetwork.tunnelIds.addAll(tunnelIds);
        return newVLinkNetwork;
    }

    public String toString(){
        StringBuffer buffer = new StringBuffer();
        buffer.append("Id: " + networkVlinkId + "\n");
        buffer.append("Endpoints: \n");
        for (ConnectPoint c: networkVlinkEndpoints){
            buffer.append(" -" + c + "\n"); 
        }
        buffer.append("Tunnel Ids: \n");
        for (Long l: tunnelIds){
            buffer.append(" -" + l + "\n"); 
        }
        return buffer.toString();
    }

}
