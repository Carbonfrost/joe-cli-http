<!-- Copyright 2026 The Joe-cli Authors. All rights reserved.
     Use of this source code is governed by a BSD-style
     license that can be found in the LICENSE file.
-->

# Test Suite

This directory contains an integration test suite for the CLI commands, implemented using **Brat**.

## Requirements

Make sure you have the following installed:

* `bash` (v4+ recommended)
* `brat` - [Brutal Runner for Automated Tests (BRAT)](https://codeberg.org/sstephenson/brat)

## Running the Tests

From the project root, run:

```sh
make test
```

## Writing New Tests

To add a new case, use one of the helpers:

```bash
run_case "description" \
  'template' \
  'expected output' \
  -T var=value
```

Or for JSON input:

```bash
run_case_stdin "description" \
  '{ "key": "value" }' \
  'template' \
  'expected output'
```
