package org.l2sm.vlinks.api;
import org.onosproject.net.Path;
import org.onosproject.net.DefaultPath;
import org.onosproject.net.ConnectPoint;

public interface IDCOVLinkService {

    public void createVLinkNetwork(String networkVlinkId, ConnectPoint networkVlinkFromEndpoint, ConnectPoint networkVlinkToEndpoint, String[] vLinkPath) throws IDCOVLinkServiceException;

    public void deleteVLinkNetwork(String networkVlinkId) throws IDCOVLinkServiceException;

    public VLinkNetwork getVLinkNetwork(String networkVlinkId) throws IDCOVLinkServiceException;

}
