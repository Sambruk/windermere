/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2024 Föreningen Sambruk
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.

 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.

 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Sambruk/windermere/windermere"
)

func main() {
	var storageTypep = flag.String("storagetype", "", "e.g. sqlite or sqlserver")
	var storageSourcep = flag.String("storagesource", "", "data source string")
	var downgradeTop = flag.Int("downgradeto", 0, "database version to downgrade to")

	flag.Parse()

	if *storageTypep == "" || *storageSourcep == "" || *downgradeTop == 0 {
		fmt.Fprintf(os.Stderr, "Usage: downgrade -storagetype <type> -storagesource <source> -downgradeto <version>\n")
		return
	}

	storageType := *storageTypep
	storageSource := *storageSourcep
	downgradeTo := *downgradeTop

	err := windermere.DowngradeDBSchema(storageType, storageSource, downgradeTo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to downgrade database schema: %s\n", err.Error())
		return
	}
	fmt.Printf("Successfully downgraded database schema to version %d\n", downgradeTo)
}
