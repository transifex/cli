package config

import (
    "testing"
)

func TestGetActiveHost(t *testing.T) {
    cfg := Config{
        Root: &RootConfig{
            Hosts: []Host{
                {Name: "aaa", RestHostname: "AAA"},
                {Name: "bbb", RestHostname: "BBB"},
            },
        },
        Local: &LocalConfig{
            Host: "aaa",
        },
    }

    activeHost := cfg.GetActiveHost()
    if activeHost != &cfg.Root.Hosts[0] {
        t.Errorf("Found wrong host '%s', expected '{aaa AAA}'", activeHost)
    }
}

func TestFind(t *testing.T) {
    cfg := Config{
        Local: &LocalConfig{
            Resources: []Resource{
                {ProjectSlug: "aaa", ResourceSlug: "bbb"},
                {ProjectSlug: "ccc", ResourceSlug: "ddd"},
            },
        },
    }

    resource := cfg.FindResource("aaa.bbb")
    if resource != &cfg.Local.Resources[0] {
        t.Errorf(
            "Got wrong resource %s, expected %s",
            *resource,
            cfg.Local.Resources[0],
        )
    }

    resource = cfg.FindResource("ccc.ddd")
    if resource != &cfg.Local.Resources[1] {
        t.Errorf(
            "Got wrong resource %s, expected %s",
            *resource,
            cfg.Local.Resources[0],
        )
    }

    resource = cfg.FindResource("something else")
    if resource != nil {
        t.Errorf("Got wrong resource %s, expected nil", *resource)
    }
}
