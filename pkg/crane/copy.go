// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crane

import (
	"log"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdCopy()) }

// NewCmdCopy creates a new cobra.Command for the copy subcommand.
func NewCmdCopy() *cobra.Command {
	return &cobra.Command{
		Use:     "copy",
		Aliases: []string{"cp"},
		Short:   "Efficiently copy a remote image from src to dst",
		Args:    cobra.ExactArgs(2),
		Run:     doCopy,
	}
}

func doCopy(_ *cobra.Command, args []string) {
	src, dst := args[0], args[1]
	srcRef, err := name.ParseReference(src, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing reference %q: %v", src, err)
	}
	log.Printf("Pulling %v", srcRef)

	dstRef, err := name.ParseReference(dst, name.WeakValidation)
	if err != nil {
		log.Fatalf("parsing reference %q: %v", dst, err)
	}
	log.Printf("Pushing %v", dstRef)

	dstAuth, err := authn.DefaultKeychain.Resolve(dstRef.Context().Registry)
	if err != nil {
		log.Fatalf("getting creds for %q: %v", dstRef, err)
	}

	desc, err := remote.Get(srcRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		log.Fatalf("fetching image %q: %v", srcRef, err)
	}

	switch desc.MediaType {
	case types.OCIImageIndex, types.DockerManifestList:
		// Handle indexes separately.
		if err := copyIndex(desc, dstRef, dstAuth); err != nil {
			log.Fatalf("failed to copy index: %v", err)
		}
	default:
		// Assume anything else is an image, since some registries don't set mediaTypes properly.
		if err := copyImage(desc, dstRef, dstAuth); err != nil {
			log.Fatalf("failed to copy image: %v", err)
		}
	}
}

func copyImage(desc *remote.Descriptor, dstRef name.Reference, dstAuth authn.Authenticator) error {
	img, err := desc.Image()
	if err != nil {
		return err
	}
	return remote.Write(dstRef, img, dstAuth, http.DefaultTransport)
}

func copyIndex(desc *remote.Descriptor, dstRef name.Reference, dstAuth authn.Authenticator) error {
	idx, err := desc.ImageIndex()
	if err != nil {
		return err
	}
	return remote.WriteIndex(dstRef, idx, dstAuth, http.DefaultTransport)
}
