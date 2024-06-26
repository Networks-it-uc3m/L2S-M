package org.l2sm.vlinks.app;

import java.util.Collection;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

import java.util.stream.Collectors;

import org.l2sm.vlinks.api.VLinkNetwork;
import org.onlab.packet.MacAddress;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.intent.Key;

// TODO: FIX Warnings depreacted library
import com.google.common.collect.HashBasedTable;
import com.google.common.collect.Table;

import org.slf4j.Logger;

public class IDCOVLinkDatabase {


    HashMap<String, VLinkNetwork> vlinkNetworks = new HashMap<>();
    HashMap<String, Key> mainIntents = new HashMap<>();
    HashMap<String, HashSet<Key>> hostIntents = new HashMap<>();

    Map<ConnectPoint, String> portVLinkNetworks = Collections.synchronizedMap( new HashMap<>());
    HashMap<ConnectPoint, Long> portTunnelIds = new HashMap<>();
    Table<String, MacAddress, ConnectPoint> macTable = HashBasedTable.create();

    Logger log;


    public IDCOVLinkDatabase(Logger log){
        this.log = log;
    }

    class CustomLock {

        int accesses;
        Lock lock;

        public CustomLock(){
            accesses = 0;
            lock = new ReentrantLock();
        }
    }

    Lock mainLock = new ReentrantLock();
    HashMap<String, CustomLock> vlinkNetworkLocks = new HashMap<>();

    public void lockVLinkNetwork(String networkVlinkId){
        CustomLock vlinkNetworkLock;
        mainLock.lock();
        vlinkNetworkLock = vlinkNetworkLocks.get(networkVlinkId);
        if (vlinkNetworkLock == null){
            vlinkNetworkLock = new CustomLock();
            vlinkNetworkLocks.put(networkVlinkId, vlinkNetworkLock);
        }
        vlinkNetworkLock.accesses += 1;
        mainLock.unlock();
        vlinkNetworkLock.lock.lock();
    }

    public void unlockVLinkNetwork(String networkVlinkId){
        CustomLock vlinkNetworkLock;
        mainLock.lock();
        vlinkNetworkLock = vlinkNetworkLocks.get(networkVlinkId);
        if (vlinkNetworkLock != null){
            vlinkNetworkLock.accesses -= 1;
            vlinkNetworkLock.lock.unlock();
            if (vlinkNetworkLock.accesses == 0){
                vlinkNetworkLocks.remove(networkVlinkId);
            }
        }
        mainLock.unlock();
 
    }

    public void addMainIntent(String networkVlinkId, Key intentKey) {
        mainIntents.put(networkVlinkId, intentKey);

    }

    public boolean vlinkNetworkExists(String networkVlinkId) {
        return vlinkNetworks.containsKey(networkVlinkId);
    }

    public VLinkNetwork getVLinkNetwork(String networkVlinkId) {
        return vlinkNetworks.get(networkVlinkId).clone();
    }

    public void deleteVLinkNetwork(String networkVlinkId) {

        VLinkNetwork vlinkNetwork = vlinkNetworks.get(networkVlinkId);
        for (ConnectPoint p: vlinkNetwork.networkVlinkEndpoints){
            portVLinkNetworks.remove(p);
            portTunnelIds.remove(p);
        }
        vlinkNetworks.remove(networkVlinkId);
        mainIntents.remove(networkVlinkId);
        hostIntents.remove(networkVlinkId);
        macTable.row(networkVlinkId).clear();        
    }

    public void registerVLinkNetwork(String networkVlinkId) {
        VLinkNetwork vlinkNetwork = new VLinkNetwork();
        vlinkNetwork.networkVlinkId = networkVlinkId;
        vlinkNetworks.put(networkVlinkId,vlinkNetwork);
        hostIntents.put(networkVlinkId, new HashSet<>());

    }

    public Collection<Key> getVLinkNetworkIntents(String networkVlinkId) {
        Set<Key> vlinkNetworkIntents = new HashSet<>();
        vlinkNetworkIntents.addAll(hostIntents.get(networkVlinkId));
        Key mainKey = mainIntents.get(networkVlinkId);
        if (mainKey != null){
            vlinkNetworkIntents.add(mainIntents.get(networkVlinkId));
        }
        return vlinkNetworkIntents;
    }

    public Iterable<Key> getAllIntents() {
        Set<Key> allIntents = new HashSet<>();
        for (String id : vlinkNetworks.keySet()){
            allIntents.addAll(getVLinkNetworkIntents(id));
        }
        return allIntents;        
    }

    public void cleanVLinkDatabases() {
        vlinkNetworks.clear();
        mainIntents.clear();
        hostIntents.clear();
        portVLinkNetworks.clear();
        portTunnelIds.clear();
        macTable.clear();
    }

    public void addPortToVLinkNetwork(String networkVlinkId, ConnectPoint networkVlinkEndpoint, Long tunnelId) {

        VLinkNetwork vlinkNetwork = vlinkNetworks.get(networkVlinkId);
        vlinkNetwork.networkVlinkEndpoints.add(networkVlinkEndpoint);
        vlinkNetwork.tunnelIds.add(tunnelId);
        portVLinkNetworks.put(networkVlinkEndpoint, networkVlinkId);
        portTunnelIds.put(networkVlinkEndpoint, tunnelId);
    }

    public Collection<ConnectPoint> getPortsOfVLinkNetworkGivenPort(ConnectPoint heardPort) {
        String networkVlinkId = getVLinkNetworkIdForPort(heardPort);
        if (networkVlinkId == null){
            return Collections.emptySet();
        }
        VLinkNetwork vlinkNetwork = vlinkNetworks.get(networkVlinkId);
        return vlinkNetwork.networkVlinkEndpoints.stream().filter(p -> !p.equals(heardPort)).collect(Collectors.toSet());
    }

    
    public String getVLinkNetworkIdForPort(ConnectPoint hostLocation) {
        return portVLinkNetworks.get(hostLocation);
    }

    public Long getTunnelIdOfPort(ConnectPoint hostLocation) {
        return portTunnelIds.get(hostLocation);
    }

    public ConnectPoint getHostLocation(String mscsId, MacAddress macAddress) {
        return macTable.get(mscsId, macAddress);
    }

    public void addIntentToVLinkNetwork(String mscsId, Key intentKey) {
        hostIntents.get(mscsId).add(intentKey);
    }

    public void setHostLocation(String mscsId, MacAddress macAddress, ConnectPoint hostLocation) {
        macTable.put(mscsId, macAddress, hostLocation);
    }

}
