/*
 * Copyright (c) 2020 PANTHEON.tech s.r.o. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"

	"github.com/ligato/cn-infra/agent"
	"github.com/ligato/cn-infra/logging"
	"go.cdnf.io/cnf-nsm/cmd/nsm-agent-linux/app"
)

const logo = "    _   _______ __  ___   ___                    __ \n" +
	"   / | / / ___//  |/  /  /   | ____ ____  ____  / /_\n" +
	"  /  |/ /\\__ \\/ /|_/ /  / /| |/ __ `/ _ \\/ __ \\/ __/\n" +
	" / /|  /___/ / /  / /  / ___ / /_/ /  __/ / / / /_  \n" +
	"/_/ |_//____/_/  /_/  /_/  |_\\__, /\\___/_/ /_/\\__/  \n" +
	"                            /____/	                 %s\n"

func main() {
	if _, err := fmt.Fprintf(os.Stdout, logo, agent.BuildVersion); err != nil {
		logging.DefaultLogger.Fatal(err)
	}

	cnfAgent := app.NewAgent()
	a := agent.NewAgent(agent.AllPlugins(cnfAgent))
	if err := a.Run(); err != nil {
		logging.DefaultLogger.Fatal(err)
	}
}
