package org.l2sm.vnets.net;

import java.util.Collections;

import org.onosproject.core.ApplicationId;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.ResourceGroup;
import org.onosproject.net.intent.Intent;
import org.onosproject.net.intent.Key;

import com.google.common.base.MoreObjects;

//Based on the TwoWayP2PIntent class
public class VirtualNetworkIntent extends Intent {

    // Protected so that the serializer is able to access them
    ConnectPoint[] connectPoints;
    long[] tunnelIds;

    /**
     * Returns a new virtual link builder.
     * Mandatory fields:
     * -ingressPoint
     * -egressPoint
     * -appId
     * -tunnelId
     * An exception will be raised if any of those are not set using
     * the appropiated methods
     *
     * @return virtual link intent builder
     */

    public static VirtualNetworkIntent.Builder builder() {
        return new Builder();
    }

    /**
     * Builder for a virtual link intent
     */
    public static final class Builder extends Intent.Builder {

        ConnectPoint[] connectPoints;
        long[] tunnelIds;

        private Builder() {
            // Hide constructor
        }

        @Override
        public Builder appId(ApplicationId appId) {
            return (Builder) super.appId(appId);
        }

        @Override
        public Builder key(Key key) {
            return (Builder) super.key(key);
        }

        @Override
        public Builder priority(int priority) {
            return (Builder) super.priority(priority);
        }

        /**
         * Sets one of the ports of the virtual link
         *
         * @param one connect point
         * @return this builder
         */
        public Builder connectPoints(ConnectPoint[] connectPoints) {
            this.connectPoints = connectPoints;
            return this;
        }

        /**
         * Sets the tunnel ID to be used for the tunnel. It is a responsibility of
         * the developer to make sure the tunnelId is the allowed range for the
         * underlying
         * tunneling technology (e.g., VXLAN or GRE)
         *
         * @param tunnelId tunnel ID of the tunnel
         * @return this builder
         */
        public Builder tunnelIDs(long[] tunnelIds) {
            this.tunnelIds = tunnelIds;
            return this;
        }

        /**
         * Builds a virtual link intent from the accumulated parameters.
         *
         * @return virtual link intent
         */
        public VirtualNetworkIntent build() {
            return new VirtualNetworkIntent(
                    appId,
                    key,
                    connectPoints,
                    tunnelIds,
                    priority,
                    resourceGroup);
        }
    }

    /**
     * Creates a new virtual link intent. Not for public use
     * and should only be accesed by the virtual link intent builder
     */
    private VirtualNetworkIntent(ApplicationId appId,
            Key key,
            ConnectPoint[] connectPoints,
            long[] tunnelIds,
            int priority,
            ResourceGroup resourceGroup) {
        super(appId,
                key,
                Collections.emptyList(),
                priority,
                resourceGroup);

        this.connectPoints = connectPoints;
        this.tunnelIds = tunnelIds;
    }

    // Constructor for serializer.

    protected VirtualNetworkIntent() {
        super();
        this.connectPoints = null;
        this.tunnelIds = null;
    }

    public ConnectPoint[] connectPoints() {
        return this.connectPoints;
    }

    /**
     * Return the id of the tunnel used in the virtual link
     *
     * @return tunnel id
     */
    public long[] tunnelIds() {
        return this.tunnelIds;
    }

    @Override
    public String toString() {
        return MoreObjects.toStringHelper(getClass())
                .add("id", id())
                .add("key", key())
                .add("appId", appId())
                .add("priority", priority())
                .add("resources", resources())
                .add("tunnelIDs", tunnelIds())
                .add("resourceGroup", resourceGroup())
                .toString();
    }

}
