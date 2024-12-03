package org.l2sm.vnets.app;

import java.util.Collection;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

import java.util.stream.Collectors;

import org.l2sm.vnets.api.Network;
import org.onlab.packet.MacAddress;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.intent.Key;

import com.google.common.collect.HashBasedTable;
import com.google.common.collect.Table;

import org.slf4j.Logger;

public class IDCODatabase {


    HashMap<String, Network> networks = new HashMap<>();
    HashMap<String, Key> mainIntents = new HashMap<>();
    HashMap<String, HashSet<Key>> hostIntents = new HashMap<>();

    Map<ConnectPoint, String> portNetworks = Collections.synchronizedMap( new HashMap<>());
    HashMap<ConnectPoint, Long> portTunnelIds = new HashMap<>();
    Table<String, MacAddress, ConnectPoint> macTable = HashBasedTable.create();

    Logger log;


    public IDCODatabase(Logger log){
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
    HashMap<String, CustomLock> networkLocks = new HashMap<>();

    public void lockNetwork(String networkId){
        CustomLock networkLock;
        mainLock.lock();
        networkLock = networkLocks.get(networkId);
        if (networkLock == null){
            networkLock = new CustomLock();
            networkLocks.put(networkId, networkLock);
        }
        networkLock.accesses += 1;
        mainLock.unlock();
        networkLock.lock.lock();
    }

    public void unlockNetwork(String networkId){
        CustomLock networkLock;
        mainLock.lock();
        networkLock = networkLocks.get(networkId);
        if (networkLock != null){
            networkLock.accesses -= 1;
            networkLock.lock.unlock();
            if (networkLock.accesses == 0){
                networkLocks.remove(networkId);
            }
        }
        mainLock.unlock();
 
    }

    public void addMainIntent(String networkId, Key intentKey) {
        mainIntents.put(networkId, intentKey);

    }

    public boolean networkExists(String networkId) {
        return networks.containsKey(networkId);
    }

    public Network getNetwork(String networkId) {
        return networks.get(networkId).clone();
    }

    public void deleteNetwork(String networkId) {

        Network network = networks.get(networkId);
        for (ConnectPoint p: network.networkEndpoints){
            portNetworks.remove(p);
            portTunnelIds.remove(p);
        }
        networks.remove(networkId);
        mainIntents.remove(networkId);
        hostIntents.remove(networkId);
        macTable.row(networkId).clear();        
    }

    public void registerNetwork(String networkId) {
        Network network = new Network();
        network.networkId = networkId;
        networks.put(networkId,network);
        hostIntents.put(networkId, new HashSet<>());

    }

    public Collection<Key> getNetworkIntents(String networkId) {
        Set<Key> networkIntents = new HashSet<>();
        networkIntents.addAll(hostIntents.get(networkId));
        Key mainKey = mainIntents.get(networkId);
        if (mainKey != null){
            networkIntents.add(mainIntents.get(networkId));
        }
        return networkIntents;
    }

    public Iterable<Key> getAllIntents() {
        Set<Key> allIntents = new HashSet<>();
        for (String id : networks.keySet()){
            allIntents.addAll(getNetworkIntents(id));
        }
        return allIntents;        
    }

    public void cleanDatabases() {
        networks.clear();
        mainIntents.clear();
        hostIntents.clear();
        portNetworks.clear();
        portTunnelIds.clear();
        macTable.clear();
    }

    public void addPortToNetwork(String networkId, ConnectPoint networkEndpoint, Long tunnelId) {

        Network network = networks.get(networkId);
        network.networkEndpoints.add(networkEndpoint);
        network.tunnelIds.add(tunnelId);
        portNetworks.put(networkEndpoint, networkId);
        portTunnelIds.put(networkEndpoint, tunnelId);
    }

    public Collection<ConnectPoint> getPortsOfNetworkGivenPort(ConnectPoint heardPort) {
        String networkId = getNetworkIdForPort(heardPort);
        if (networkId == null){
            return Collections.emptySet();
        }
        Network network = networks.get(networkId);
        return network.networkEndpoints.stream().filter(p -> !p.equals(heardPort)).collect(Collectors.toSet());
    }

    
    public String getNetworkIdForPort(ConnectPoint hostLocation) {
        return portNetworks.get(hostLocation);
    }

    public Long getTunnelIdOfPort(ConnectPoint hostLocation) {
        return portTunnelIds.get(hostLocation);
    }

    public ConnectPoint getHostLocation(String mscsId, MacAddress macAddress) {
        return macTable.get(mscsId, macAddress);
    }

    public void addIntentToNetwork(String mscsId, Key intentKey) {
        hostIntents.get(mscsId).add(intentKey);
    }

    public void setHostLocation(String mscsId, MacAddress macAddress, ConnectPoint hostLocation) {
        macTable.put(mscsId, macAddress, hostLocation);
    }

}
