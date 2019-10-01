// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// see the license for the specific language governing permissions and
// limitations under the license.

/*
The mako command gives users of external mako (go/mako for more information) the
ability to accomplish tasks via the command line.

Execute with no params to see usage message. Providing the incorrect arguments
to a command will result in the command usage being displayed.

All queries are done by primary key which ensures strong consistency.

Due to retry logic both inside mako clients and on the server
invalid queries (eg. for a non-existent key) can take a while. If you'd like to
see the output from the query library add these flags to your query before the
command. Example:
  $ mako -vmodule=google3storage=2 --alsologtostderr <command> ...


*/
package main

import (
	"context"
	"flag"
	"os"

	log "github.com/golang/glog"
	"github.com/google/mako/clients/go/storage/mako"
	"github.com/google/mako/internal/tools/cli/lib"
	"github.com/google/subcommands"
)

func main() {
	flag.Parse()
	storage, err := mako.New()
	if err != nil {
		log.Fatalf("failure instantiating mako client: %v", err)
	}
	ret := lib.Run(context.Background(), storage, os.Stdout, os.Stderr, subcommands.DefaultCommander)
	os.Exit(int(ret))
}
