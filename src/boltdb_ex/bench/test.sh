#!/bin/bash
#BenchmarkGobEncode-8          1000000      2175 ns/op
# BenchmarkJsonEncode-8         300000      5119 ns/op
# BenchmarkGobDecode-8          500000      2370 ns/op
# BenchmarkJsonDecode-8         200000     11313 ns/op

go test -bench .
