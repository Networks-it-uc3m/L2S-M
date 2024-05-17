package com.l2sm.database.entity;

import javax.persistence.*;

@Entity
@Table(name = "switches")
public class Switch {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Integer id;

    @Column(name = "node_name", nullable = false, length = 255)
    private String nodeName;

    @Column(name = "openflowId", columnDefinition = "TEXT")
    private String openflowId;

    @Column(name = "ip", length = 15)
    private String ip;

    // Getters and setters
}
