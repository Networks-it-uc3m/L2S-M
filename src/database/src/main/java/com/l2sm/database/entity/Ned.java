package com.l2sm.database.entity;

import javax.persistence.*;

@Entity
@Table(name = "neds")
public class Ned {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Integer id;

    @Column(name = "node_name", nullable = false, length = 255)
    private String nodeName;

    @Column(name = "provider", nullable = false, length = 255)
    private String provider;

    @Column(name = "openflowId", columnDefinition = "TEXT")
    private String openflowId;

    @Column(name = "ip", length = 15)
    private String ip;

    // Getters and setters

    public Integer getId() {
        return id;
    }

    public void setId(Integer id) {
        this.id = id;
    }

    public String getNodeName() {
        return nodeName;
    }

    public void setNodeName(String nodeName) {
        this.nodeName = nodeName;
    }

    public String getProvider() {
        return provider;
    }

    public void setProvider(String provider) {
        this.provider = provider;
    }

    public String getOpenflowId() {
        return openflowId;
    }

    public void setOpenflowId(String openflowId) {
        this.openflowId = openflowId;
    }

    public String getIp() {
        return ip;
    }

    public void setIp(String ip) {
        this.ip = ip;
    }
}
