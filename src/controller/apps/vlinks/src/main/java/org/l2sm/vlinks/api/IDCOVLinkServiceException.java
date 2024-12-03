package org.l2sm.vlinks.api;

public class IDCOVLinkServiceException extends Exception{

    private String message;

    public IDCOVLinkServiceException(String message){
        this.message = message;
    }

    public String getMessage(){
        return this.message;
    }    

    @Override
    public String toString(){
        return "ERROR: " + this.message;
    }
    
}
