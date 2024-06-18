package com.l2sm.database.entity;

import javax.persistence.*;
import java.util.Set;

@Entity
@Table(name = "link")
public class Link {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Integer id;

    @Column(name = "link_name", nullable = false, length = 255)
    private String linkName;

    @ManyToOne
    @JoinColumn(name = "end_A")
    private Switch endA;

    @ManyToOne
    @JoinColumn(name = "end_B")
    private Switch endB;

    @ManyToMany
    Set<Path> linkedPaths;
    // Getters and setters

    public Integer getId() {
        return id;
    }

    public void setId(Integer id) {
        this.id = id;
    }

    public String getLinkName() {
        return linkName;
    }

    public void setLinkName(String linkName) {
        this.linkName = linkName;
    }

    public Switch getEndA() {
        return endA;
    }

    public void setEndA(Switch endA) {
        this.endA = endA;
    }

    public Switch getEndB() {
        return endB;
    }

    public void setEndB(Switch endB) {
        this.endB = endB;
    }

    public Set<Path> getLinkedPaths() {
        return linkedPaths;
    }

    public void setLinkedPaths(Set<Path> linkedPaths) {
        this.linkedPaths = linkedPaths;
    }
}
