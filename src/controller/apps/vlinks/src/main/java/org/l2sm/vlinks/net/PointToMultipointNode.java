package org.l2sm.vlinks.net;

import java.util.Collection;
import java.util.HashMap;
import java.util.Map;

import org.onosproject.net.ConnectPoint;

public class PointToMultipointNode {

    private ConnectPoint value;

    private HashMap<ConnectPoint, PointToMultipointNode> children = new HashMap<>();

    public PointToMultipointNode(ConnectPoint cp) {
        this.value = cp;
    }

    public ConnectPoint getRootPort() {
        return value;
    }

    public PointToMultipointNode addChild(ConnectPoint cp, ConnectPoint next) {
        PointToMultipointNode nextNode = next == null ? null : new PointToMultipointNode(next);
        children.put(cp, nextNode);
        return nextNode;
    }

    public Collection<PointToMultipointNode> getChildrenNodes() {
        return children.values();
    }

    public Map<ConnectPoint, PointToMultipointNode> getChildren() {
        return children;
    }

    public PointToMultipointNode getChild(ConnectPoint cp) {
        return children.get(cp);
    }

    public String toString() {
        StringBuffer buffer = new StringBuffer(this.value.toString() + "-> ");
        children.keySet().forEach(child -> buffer.append(child.toString() + ", "));
        children.forEach((port, child) -> {
            buffer.append(port.toString() + " ==> " + (child == null ? "" : child.toString()));
        });
        return buffer.toString();
    }

}
