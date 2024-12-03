package org.l2sm.vnets.rest;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.lang.annotation.Annotation;
import java.lang.reflect.Type;

import javax.ws.rs.Consumes;
import javax.ws.rs.Produces;
import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;
import javax.ws.rs.core.Response;
import javax.ws.rs.ext.MessageBodyReader;
import javax.ws.rs.ext.MessageBodyWriter;
import javax.ws.rs.ext.Provider;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;


//We could use jackson jax-rs provider but we want to have more control
@Provider
@Produces({"application/yaml",MediaType.APPLICATION_JSON,})
@Consumes({"application/yaml",MediaType.APPLICATION_JSON})
public class ObjectProvider implements MessageBodyWriter<Object>, MessageBodyReader<Object>  {

    private static final ObjectMapper jsonObjectMapper = new ObjectMapper();
    private static final ObjectMapper yamlObjectMapper = new ObjectMapper(new YAMLFactory());

    static{
        jsonObjectMapper.configure(SerializationFeature.FAIL_ON_EMPTY_BEANS, false);
        jsonObjectMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, true);
        jsonObjectMapper.configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, true);
        yamlObjectMapper.configure(SerializationFeature.FAIL_ON_EMPTY_BEANS, false);
        yamlObjectMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, true);
        yamlObjectMapper.configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, true);
    }
    
    @Override
    public boolean isReadable(Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType) {
        return true;
    }

    @Override
    public Object readFrom(Class<Object> type, Type genericType, Annotation[] annotations, MediaType mediaType,
            MultivaluedMap<String, String> httpHeaders, InputStream entityStream)
            throws IOException, WebApplicationException {

        ObjectMapper mapper = null;
        Object o = null;
        if (mediaType.equals(MediaType.APPLICATION_JSON)){
            mapper = jsonObjectMapper;
        }else{
            mapper = yamlObjectMapper;
        }
        try{
            o = mapper.readValue(entityStream, type);
        }catch(JsonMappingException e){
            throw new WebApplicationException(e.getLocalizedMessage(),Response.Status.BAD_REQUEST);
        } catch (Exception e){
            throw new WebApplicationException("Other exception",Response.Status.BAD_REQUEST);
        }
        
        return o;
    }

    @Override
    public boolean isWriteable(Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType) {
        return true;
    }

    @Override
    public void writeTo(Object t, Class<?> type, Type genericType, Annotation[] annotations, MediaType mediaType,
            MultivaluedMap<String, java.lang.Object> httpHeaders, OutputStream entityStream)
            throws IOException, WebApplicationException {

        ObjectMapper mapper = null;

        if (mediaType.equals(MediaType.APPLICATION_JSON)){
            mapper = jsonObjectMapper;
        }else{
            mapper = yamlObjectMapper;
        }
        mapper.writeValue(entityStream,t);

    }


}
