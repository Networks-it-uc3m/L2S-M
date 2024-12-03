package org.l2sm.vlinks.dto;

import java.util.ArrayList;

public class NetworkDTO {

    private String networkId;

    private ArrayList<String> networkEndpoints;

    private ArrayList<Long> tunnelList;

    public String getNetworkId() {
        return networkId;
    }

    public void setNetworkId(String networkId) {
        this.networkId = networkId;
    }

    public ArrayList<String> getNetworkEndpoints() {
        return networkEndpoints;
    }

    public void setNetworkEndpoints(ArrayList<String> networkEndpoints) {
        this.networkEndpoints = networkEndpoints;
    }

    public ArrayList<Long> getTunnelList() {
        return tunnelList;
    }

    public void setTunnelList(ArrayList<Long> tunnelList) {
        this.tunnelList = tunnelList;
    }

   
    
}
