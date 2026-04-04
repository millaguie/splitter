package tor

import (
	"context"
	"testing"

	"github.com/user/splitter/internal/process"
)

func TestManager_GetInstanceInfos(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	countries := []string{"{US}", "{DE}"}

	mgr.CreateFromVersion(v, countries)

	infos := mgr.GetInstanceInfos()
	if len(infos) != 4 {
		t.Fatalf("GetInstanceInfos() returned %d, want 4", len(infos))
	}

	expected := []InstanceInfo{
		{ID: 0, Country: "{US}"},
		{ID: 1, Country: "{US}"},
		{ID: 2, Country: "{DE}"},
		{ID: 3, Country: "{DE}"},
	}

	for i, info := range infos {
		if info.ID != expected[i].ID {
			t.Errorf("infos[%d].ID = %d, want %d", i, info.ID, expected[i].ID)
		}
		if info.Country != expected[i].Country {
			t.Errorf("infos[%d].Country = %q, want %q", i, info.Country, expected[i].Country)
		}
	}
}

func TestManager_GetInstanceInfos_Empty(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	infos := mgr.GetInstanceInfos()
	if len(infos) != 0 {
		t.Errorf("GetInstanceInfos() returned %d, want 0", len(infos))
	}
}

func TestManager_RotateInstance_NotFound(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	ctx := context.Background()
	err := mgr.RotateInstance(ctx, 999, "{FR}")
	if err == nil {
		t.Error("RotateInstance(999) expected error, got nil")
	}
}

func TestManager_RotateInstance_CountryUpdatedBeforeStart(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}"})

	ctx := context.Background()
	_ = mgr.RotateInstance(ctx, 0, "{FR}")

	infos := mgr.GetInstanceInfos()
	if len(infos) != 1 {
		t.Fatalf("GetInstanceInfos() returned %d instances, want 1", len(infos))
	}
	if infos[0].Country != "{FR}" {
		t.Errorf("Country = %q, want %q after RotateInstance", infos[0].Country, "{FR}")
	}
}

func TestManager_RotateInstance_ByID(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}", "{DE}", "{FR}"})

	ctx := context.Background()
	_ = mgr.RotateInstance(ctx, 1, "{GB}")

	infos := mgr.GetInstanceInfos()
	if len(infos) != 3 {
		t.Fatalf("GetInstanceInfos() returned %d instances, want 3", len(infos))
	}

	if infos[0].Country != "{US}" {
		t.Errorf("instance 0 Country = %q, want {US} (unchanged)", infos[0].Country)
	}
	if infos[1].Country != "{GB}" {
		t.Errorf("instance 1 Country = %q, want {GB} (rotated)", infos[1].Country)
	}
	if infos[2].Country != "{FR}" {
		t.Errorf("instance 2 Country = %q, want {FR} (unchanged)", infos[2].Country)
	}
}
