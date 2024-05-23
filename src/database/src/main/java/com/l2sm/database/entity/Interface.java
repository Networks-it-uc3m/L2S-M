package com.l2sm.database.entity;

import javax.persistence.*;


@Entity
@Table(name = "interfaces")
public class Interface {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Integer id;

    @Column(name = "name", length = 255)
    private String name;

    @Column(name = "pod", length = 255)
    private String pod;

    @ManyToOne
    @JoinColumn(name = "switch_id")
    private Switch switchEntity;

    @ManyToOne
    @JoinColumn(name = "ned_id")
    private Ned ned;

    @ManyToOne
    @JoinColumn(name = "network_id")
    private Network network;

    @ManyToOne
    @JoinColumn(name = "link_id")
    private Link link;

    public Integer getId() {
        return id;
    }

    public void setId(Integer id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getPod() {
        return pod;
    }

    public void setPod(String pod) {
        this.pod = pod;
    }

    public Switch getSwitchEntity() {
        return switchEntity;
    }

    public void setSwitchEntity(Switch switchEntity) {
        this.switchEntity = switchEntity;
    }

    public Ned getNed() {
        return ned;
    }

    public void setNed(Ned ned) {
        this.ned = ned;
    }

    public Network getNetwork() {
        return network;
    }

    public void setNetwork(Network network) {
        this.network = network;
    }

    public Link getLink() {
        return link;
    }

    public void setLink(Link link) {
        this.link = link;
    }
// Getters and setters
}
