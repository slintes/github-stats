# Github Stats

A tool for gathering some stats about your github activities.
Atm it will report

- how many PRs of you are merged
- how many PRs are merged by you (also takes `/approve` comments into account)
- how many PRs are reviewed by you (via gitgub review functionality or just normal comment)

PRs falling into one category are not counted for the ones anymore in the same month, so e.g. PRs merged will not be counted as PRs reviewed as well.

# Usage

`go run stats -user <githubUser> -token <githubToken> -months <nrMonths>  -repositories <repositories>`

- `githubUser` = the user for which stats are collected
- `githubToken` = a token for acessing the github API
- `nrMonths` = the nr of month to go backwards
- `repositories` = a commas separated list of repositories in org/name format

# Development

TODO

# License

Github stats is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt).

    Copyright 2019

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.