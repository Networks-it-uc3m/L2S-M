package org.l2sm.vlinks.net;

import java.util.Collections;

import org.onosproject.core.ApplicationId;
import org.onosproject.net.ConnectPoint;
import org.onosproject.net.ResourceGroup;
import org.onosproject.net.intent.Intent;
import org.onosproject.net.Path;
import org.onosproject.net.DefaultPath;
import org.onosproject.net.intent.Key;

import com.google.common.base.MoreObjects;

//Based on the TwoWayP2PIntent class
public class VLinkPathIntent extends Intent {

    // Protected so that the serializer is able to access them
    ConnectPoint one;
    ConnectPoint two;
    String[] path;
    long tunnelId;

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

    public static VLinkPathIntent.Builder builder() {
        return new Builder();
    }

    /**
     * Builder for a virtual link intent
     */
    public static final class Builder extends Intent.Builder {

        ConnectPoint one = null;
        ConnectPoint two = null;
        String[] path; //no creo que esto sea así revisar
        long tunnelId;

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
        public Builder one(ConnectPoint one) {
            this.one = one;
            return this;
        }

        /**
         * Sets one of the ports of the virtual link
         *
         * @param two connect point
         * @return this builder
         */
        public Builder two(ConnectPoint two) {
            this.two = two;
            return this;
        }

        /**
         * Sets the  vnis Id of the Path
         *
         * @param path Path
         * @return this builder
         */
        public Builder path(String[] path) {
            this.path = path;
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
        public Builder tunnelID(long tunnelId) {
            this.tunnelId = tunnelId;
            return this;
        }


        /**
         * Builds a virtual link intent from the accumulated parameters.
         *
         * @return virtual link intent
         */
        public VLinkPathIntent build() {
            return new VLinkPathIntent(
                    appId,
                    key,
                    one,
                    two,
                    path,
                    tunnelId,
                    priority,
                    resourceGroup);
        }
    }

    /**
     * Creates a new virtual link intent. Not for public use
     * and should only be accesed by the virtual link intent builder
     */
    private VLinkPathIntent(ApplicationId appId,
            Key key,
            ConnectPoint one,
            ConnectPoint two,
            String[] path,
            long tunnelId,
            int priority,
            ResourceGroup resourceGroup) {
        super(appId,
                key,
                Collections.emptyList(),
                priority,
                resourceGroup);

        this.one = one;
        this.two = two;
        this.path = path;
        this.tunnelId = tunnelId;
    }

    // Constructor for serializer.

    protected VLinkPathIntent() {
        super();
        this.one = null;
        this.two = null;
        this.path = null;
        this.tunnelId = -1;
    }

    /**
     * Returns one of the ports of the virtual link
     *
     * @return one of the ports
     */
    public ConnectPoint one() {
        return one;
    }

    /**
     * Returns two of the ports of the virtual link
     *
     * @return one of the ports
     */
    public ConnectPoint two() {
        return two;
    }

    /**
     * Return the id of the tunnel used in the virtual link
     *
     * @return tunnel id
     */
    public long tunnelId() {
        return tunnelId;
    }


    /**
     * Return the path used in the virtual link
     *
     * @return path
     */
    public String[] path() {
        return path;
    }


    @Override
    public String toString() {
        return MoreObjects.toStringHelper(getClass())
                .add("id", id())
                .add("key", key())
                .add("appId", appId())
                .add("priority", priority())
                .add("resources", resources())
                .add("one", one())
                .add("two", two())
                .add("path", path())
                .add("tunnelID", tunnelId())
                .add("resourceGroup", resourceGroup())
                .toString();
    }

}
