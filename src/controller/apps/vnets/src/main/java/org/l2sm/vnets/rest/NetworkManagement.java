/*
 * Copyright 2022-present Open Networking Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.l2sm.vnets.rest;


import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.l2sm.vnets.api.IDCOService;
import org.l2sm.vnets.api.IDCOServiceException;
import org.l2sm.vnets.api.Network;
import org.l2sm.vnets.dto.NetworkDTO;
import org.onosproject.net.ConnectPoint;
import org.onosproject.rest.AbstractWebResource;


/**
 * The Network MAnagement
 */
@Path("/")
public class NetworkManagement extends AbstractWebResource {


   private IDCOService idcoService = get(IDCOService.class);

    /**
     * Implementation of get a specific Network
     *
     * @return 200 OK
     */
    @GET
    @Path("/{networkId}")
    @Produces({"application/yaml",MediaType.APPLICATION_JSON})
    public Response getNetworkById(@PathParam("networkId") String networkId) throws Exception {
        
        Network network = idcoService.getVirtualNetwork(networkId);

     
        if(network==null) return Response.status(Status.NOT_FOUND).build();

        return Response.status(Status.OK).entity(network).build();
    }



     /**
     * The idco is up and running
     *
     * @return 200 OK
     */
    @GET
    @Path("/status")
    public Response getStatus() throws Exception {
            
        return Response.status(Status.OK).build();
    }

    /**
     * Implementation of the Create Network
     *
     * 
     * @return 200 OK
     */
    @POST
    @Consumes({"application/yaml",MediaType.APPLICATION_JSON})
    public Response createNetwork(NetworkDTO networkDTO) throws Exception {        

        idcoService.createVirtualNetwork(networkDTO.getNetworkId());

        return Response.status(Status.NO_CONTENT).build();
    }


    /**
     * Implementation of the Terminate Network operation
     *
     * @return 200 OK
     */
    @DELETE
    @Path("/{networkId}")
    public Response deleteNetwork(@PathParam("networkId") String networkId) throws Exception {

        idcoService.deleteVirtualNetwork(networkId);
    
        return Response.status(Status.NO_CONTENT).build();
    }

    /**
     * Implementation of the Add Port instruction
     * 
     * @return 204 CREATED
     */
    @POST
    @Path("/port")
    @Consumes({"application/yaml", MediaType.APPLICATION_JSON})
    public Response createPort(NetworkDTO networkDTO) throws Exception {

        String networkId = networkDTO.getNetworkId();
        try {
            networkDTO.getNetworkEndpoints().forEach((networkEndpoint) -> {
                try {
                    idcoService.addPort(networkId, ConnectPoint.deviceConnectPoint(networkEndpoint));
                } catch (IDCOServiceException e) {
                    throw new WebApplicationException(Response.status(Status.CONFLICT).build());
                }
            });
        } catch (WebApplicationException e) {
            return Response.status(Status.CONFLICT).build();
        }

        return Response.status(Status.NO_CONTENT).build();
    }






}