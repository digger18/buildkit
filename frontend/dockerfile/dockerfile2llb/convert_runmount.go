// +build dfrunmount dfextall

package dockerfile2llb

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func detectRunMount(cmd *command, dispatchStatesByName map[string]*dispatchState, allDispatchStates []*dispatchState) bool {
	if c, ok := cmd.Command.(*instructions.RunCommand); ok {
		mounts := instructions.GetMounts(c)
		sources := make([]*dispatchState, len(mounts))
		for i, mount := range mounts {
			if mount.From == "" && mount.Type == "cache" {
				mount.From = emptyImageName
			}
			from := mount.From
			if from == "" {
				continue
			}
			stn, ok := dispatchStatesByName[strings.ToLower(from)]
			if !ok {
				stn = &dispatchState{
					stage:        instructions.Stage{BaseName: from},
					deps:         make(map[*dispatchState]struct{}),
					unregistered: true,
				}
			}
			sources[i] = stn
		}
		cmd.sources = sources
		return true
	}

	return false
}

func dispatchRunMounts(d *dispatchState, c *instructions.RunCommand, sources []*dispatchState, opt dispatchOpt) []llb.RunOption {
	var out []llb.RunOption
	mounts := instructions.GetMounts(c)

	for i, mount := range mounts {
		if mount.From == "" && mount.Type == "cache" {
			mount.From = emptyImageName
		}
		st := opt.buildContext
		if mount.From != "" {
			st = sources[i].state
		}
		var mountOpts []llb.MountOption
		if mount.ReadOnly {
			mountOpts = append(mountOpts, llb.Readonly)
		}
		if mount.Type == "cache" {
			mountOpts = append(mountOpts, llb.AsPersistentCacheDir(opt.cacheIDNamespace+"/"+mount.CacheID))
		}
		if src := path.Join("/", mount.Source); src != "/" {
			mountOpts = append(mountOpts, llb.SourcePath(src))
		}
		out = append(out, llb.AddMount(path.Join("/", mount.Target), st, mountOpts...))

		d.ctxPaths[path.Join("/", filepath.ToSlash(mount.Source))] = struct{}{}
	}
	return out
}
