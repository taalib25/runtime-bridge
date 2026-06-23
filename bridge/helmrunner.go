package main

import (
	"context"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

// HelmRunner is the seam between bridge lifecycle code (CreateInstance, UpdateInstance,
// DeleteInstance, RollbackInstance, lookupRelease, ListInstances) and the Helm SDK's
// action package. realHelmRunner wraps the real SDK calls with the exact same options
// each call site set directly before this seam existed — no behavior change. Tests
// substitute FakeHelmRunner so lifecycle logic can be exercised without a live
// Kubernetes API server or Helm release storage backend.
type HelmRunner interface {
	Install(ctx context.Context, cfg *action.Configuration, opts InstallOptions, chrt *chart.Chart, values map[string]any) (*release.Release, error)
	Upgrade(ctx context.Context, cfg *action.Configuration, opts UpgradeOptions, releaseName string, chrt *chart.Chart, values map[string]any) (*release.Release, error)
	Uninstall(cfg *action.Configuration, opts UninstallOptions, releaseName string) error
	Rollback(cfg *action.Configuration, opts RollbackOptions, releaseName string) error
	List(cfg *action.Configuration, opts ListOptions) ([]*release.Release, error)
}

type InstallOptions struct {
	ReleaseName     string
	Namespace       string
	CreateNamespace bool
	SkipCRDs        bool
	Wait            bool
}

type UpgradeOptions struct {
	Namespace string
	SkipCRDs  bool
	Wait      bool
}

type UninstallOptions struct {
	Wait           bool
	KeepHistory    bool
	IgnoreNotFound bool
}

type RollbackOptions struct {
	Version int
	Wait    bool
	Timeout time.Duration
}

type ListOptions struct {
	All           bool
	AllNamespaces bool
	Filter        string
}

// realHelmRunner is the production HelmRunner — every field set below is exactly what
// the call sites set directly on the action.* struct before this seam existed.
type realHelmRunner struct{}

func (realHelmRunner) Install(ctx context.Context, cfg *action.Configuration, opts InstallOptions, chrt *chart.Chart, values map[string]any) (*release.Release, error) {
	install := action.NewInstall(cfg)
	install.ReleaseName = opts.ReleaseName
	install.Namespace = opts.Namespace
	install.CreateNamespace = opts.CreateNamespace
	install.SkipCRDs = opts.SkipCRDs
	install.Wait = opts.Wait
	return install.RunWithContext(ctx, chrt, values)
}

func (realHelmRunner) Upgrade(ctx context.Context, cfg *action.Configuration, opts UpgradeOptions, releaseName string, chrt *chart.Chart, values map[string]any) (*release.Release, error) {
	upgrade := action.NewUpgrade(cfg)
	upgrade.Namespace = opts.Namespace
	upgrade.SkipCRDs = opts.SkipCRDs
	upgrade.Wait = opts.Wait
	return upgrade.RunWithContext(ctx, releaseName, chrt, values)
}

func (realHelmRunner) Uninstall(cfg *action.Configuration, opts UninstallOptions, releaseName string) error {
	uninstall := action.NewUninstall(cfg)
	uninstall.Wait = opts.Wait
	uninstall.KeepHistory = opts.KeepHistory
	uninstall.IgnoreNotFound = opts.IgnoreNotFound
	_, err := uninstall.Run(releaseName)
	return err
}

func (realHelmRunner) Rollback(cfg *action.Configuration, opts RollbackOptions, releaseName string) error {
	rollback := action.NewRollback(cfg)
	rollback.Version = opts.Version
	rollback.Wait = opts.Wait
	rollback.Timeout = opts.Timeout
	return rollback.Run(releaseName)
}

func (realHelmRunner) List(cfg *action.Configuration, opts ListOptions) ([]*release.Release, error) {
	lister := action.NewList(cfg)
	lister.All = opts.All
	lister.AllNamespaces = opts.AllNamespaces
	lister.Filter = opts.Filter
	return lister.Run()
}
