package org.l2sm.vnets.api;

public class IDCOServiceException extends Exception{

    private String message;

    public IDCOServiceException(String message){
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
