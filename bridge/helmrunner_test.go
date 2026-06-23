package main

import (
	"context"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

// FakeHelmRunner is a HelmRunner that records every call and returns canned
// results, so lifecycle code (CreateInstance, UpdateInstance, DeleteInstance,
// RollbackInstance, lookupRelease, ListInstances) can be unit tested without a
// live Kubernetes API server or Helm release storage backend.
type FakeHelmRunner struct {
	InstallCalls   []FakeInstallCall
	UpgradeCalls   []FakeUpgradeCall
	UninstallCalls []FakeUninstallCall
	RollbackCalls  []FakeRollbackCall
	ListCalls      []ListOptions

	InstallResult *release.Release
	InstallErr    error
	UpgradeResult *release.Release
	UpgradeErr    error
	UninstallErr  error
	RollbackErr   error
	ListResult    []*release.Release
	ListErr       error
}

type FakeInstallCall struct {
	Opts   InstallOptions
	Values map[string]any
}

type FakeUpgradeCall struct {
	Opts        UpgradeOptions
	ReleaseName string
	Values      map[string]any
}

type FakeUninstallCall struct {
	Opts        UninstallOptions
	ReleaseName string
}

type FakeRollbackCall struct {
	Opts        RollbackOptions
	ReleaseName string
}

func (f *FakeHelmRunner) Install(_ context.Context, _ *action.Configuration, opts InstallOptions, _ *chart.Chart, values map[string]any) (*release.Release, error) {
	f.InstallCalls = append(f.InstallCalls, FakeInstallCall{Opts: opts, Values: values})
	if f.InstallErr != nil {
		return nil, f.InstallErr
	}
	if f.InstallResult != nil {
		return f.InstallResult, nil
	}
	return &release.Release{Name: opts.ReleaseName, Namespace: opts.Namespace}, nil
}

func (f *FakeHelmRunner) Upgrade(_ context.Context, _ *action.Configuration, opts UpgradeOptions, releaseName string, _ *chart.Chart, values map[string]any) (*release.Release, error) {
	f.UpgradeCalls = append(f.UpgradeCalls, FakeUpgradeCall{Opts: opts, ReleaseName: releaseName, Values: values})
	if f.UpgradeErr != nil {
		return nil, f.UpgradeErr
	}
	if f.UpgradeResult != nil {
		return f.UpgradeResult, nil
	}
	return &release.Release{Name: releaseName, Namespace: opts.Namespace}, nil
}

func (f *FakeHelmRunner) Uninstall(_ *action.Configuration, opts UninstallOptions, releaseName string) error {
	f.UninstallCalls = append(f.UninstallCalls, FakeUninstallCall{Opts: opts, ReleaseName: releaseName})
	return f.UninstallErr
}

func (f *FakeHelmRunner) Rollback(_ *action.Configuration, opts RollbackOptions, releaseName string) error {
	f.RollbackCalls = append(f.RollbackCalls, FakeRollbackCall{Opts: opts, ReleaseName: releaseName})
	return f.RollbackErr
}

func (f *FakeHelmRunner) List(_ *action.Configuration, opts ListOptions) ([]*release.Release, error) {
	f.ListCalls = append(f.ListCalls, opts)
	return f.ListResult, f.ListErr
}
