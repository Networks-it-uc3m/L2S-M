package org.l2sm.vnets.api;

import org.onosproject.net.ConnectPoint;

public interface IDCOService {

    public void createVirtualNetwork(String networkId) throws IDCOServiceException;

    public void deleteVirtualNetwork(String networkId) throws IDCOServiceException;

    public void addPort(String networkId, ConnectPoint networkEndpoint) throws IDCOServiceException;

    public Network getVirtualNetwork(String networkId) throws IDCOServiceException;

}
