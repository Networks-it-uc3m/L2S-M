package org.l2sm.vnets.rest;

import org.onlab.rest.AbstractWebApplication;

import java.util.Set;

/**
 * REST API Application
 */
public class NBInterfaceWebApplication extends AbstractWebApplication {
    @Override
    public Set<Class<?>> getClasses() {
        return getClasses(NetworkManagement.class, RESTExceptionMapper.class, ObjectProvider.class);
    }
}
