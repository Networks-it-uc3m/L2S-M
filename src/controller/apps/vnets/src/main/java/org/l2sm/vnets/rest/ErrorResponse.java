package org.l2sm.vnets.rest;

public class ErrorResponse {

    protected String errorCode;
    protected String errorDescription;


    public ErrorResponse (String error_code, String error_description){
        this.errorCode = error_code;
        this.errorDescription = error_description;
    }

    public String getErrorCode(){
        return errorCode;
    }

    public String getErrorDescription(){
        return errorDescription;
    }
    
}
