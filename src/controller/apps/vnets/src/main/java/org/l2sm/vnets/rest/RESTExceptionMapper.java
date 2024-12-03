package org.l2sm.vnets.rest;

import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;
import javax.ws.rs.ext.ExceptionMapper;
import javax.ws.rs.ext.Provider;

import org.l2sm.vnets.api.IDCOServiceException;

import com.fasterxml.jackson.core.JsonParseException;

@Provider
public class RESTExceptionMapper implements ExceptionMapper<Exception> {

    private final static String INTERNAL_ERROR_MESSAGE = "Internal error, please contact the administrator :(";
    private final static String PARSE_ERROR_MESSAGE = "Could not parse the received object";

    @Override
    public Response toResponse(Exception generalException) {

        if (generalException instanceof JsonParseException) {
            ErrorResponse errorResponse = new ErrorResponse("PARSING_ERROR", PARSE_ERROR_MESSAGE);
            return Response.status(Status.BAD_REQUEST).entity(errorResponse).build();
        }

        if (!(generalException instanceof IDCOServiceException)) {
            ErrorResponse errorResponse = new ErrorResponse("INTERNAL_ERROR", INTERNAL_ERROR_MESSAGE);
            return Response.status(Status.INTERNAL_SERVER_ERROR).entity(errorResponse).build();
        }

        IDCOServiceException exception = (IDCOServiceException) generalException;

        Status status = null;



        ErrorResponse errorResponse = new ErrorResponse(exception.toString(), exception.getMessage());

        return Response.status(status).entity(errorResponse).build();
    }

}
