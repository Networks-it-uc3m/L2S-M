package com.l2sm.database.entity;

import javax.persistence.*;
import java.util.Set;

@Entity
@Table(name = "path")
public class Path {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Integer id;

    @Column(name = "name", length = 255)
    private String name;

    @ManyToMany
    Set<Link> linkedLinks;



    // Getters and setters

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

    public Set<Link> getLinkedLinks() {
        return linkedLinks;
    }

    public void setLinkedLinks(Set<Link> linkedLinks) {
        this.linkedLinks = linkedLinks;
    }
}
