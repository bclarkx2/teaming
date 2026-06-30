# teaming

Assign small, pre-formed groups of people to larger teams — keeping every group
intact — so that team sizes land in a desired band.

## How it works

teaming applies four rules in strict priority order:

1. **Groups stay together (hard constraint).** No person is ever separated from
   their group. This rule is never violated.
2. **Minimize teams over Max.** Teams larger than `--max` are the worst outcome;
   teaming minimises how many there are.
3. **Maximize teams at or above Min.** More teams at full size is better.
4. **Minimize teams under Min.** Small teams are acceptable only as a last
   resort; teaming keeps them as few as possible.

## Installation

**Build from source** (requires Go 1.26+):

```sh
git clone https://github.com/bclarkx2/teaming.git
cd teaming
go build -o bin/teaming ./cmd/teaming
# or
make build
```

The binary is written to `bin/teaming`.

## Usage

```sh
teaming -i guests.csv -o teams.csv --min 3 --max 4
```

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--input` | `-i` | _(required)_ | Input CSV file (`person,group` header) |
| `--output` | `-o` | same as input | Output CSV file (written with `person,group,team`) |
| `--min` | | `3` | Minimum desired team size |
| `--max` | | `4` | Maximum desired team size |
| `--exact-threshold` | | `0` | Maximum number of groups for the exact optimal solver; above this the faster greedy heuristic is used. `0` uses the built-in default (12). |
| `--config` | | `teaming.yaml` | Config file path |

### Configuration precedence

Command-line flags override everything; the full order is:

```
CLI flags  >  environment variables  >  config file  >  built-in defaults
```

**Environment variables** use the prefix `TEAMING_`:

```sh
export TEAMING_INPUT=guests.csv
export TEAMING_OUTPUT=teams.csv
export TEAMING_MIN=3
export TEAMING_MAX=4
export TEAMING_EXACT_THRESHOLD=20   # optional; 0 (or unset) = built-in default (12)
teaming
```

**Config file** (`teaming.yaml` by default, or set with `--config`):

```yaml
# teaming.yaml
input:  guests.csv
output: teams.csv
min: 3
max: 4
exact-threshold: 0   # 0 = built-in default (12); set higher to force exact solver on larger inputs
```

### Input and output CSV

Input header: `person,group`  
Output header: `person,group,team`

Any existing `team` column in the input is ignored on read, so you can run
teaming in place to regenerate team assignments (idempotent):

```sh
teaming -i teams.csv   # reads person + group, overwrites with new team column
```

## Example: trivia night

You run a pub trivia night. Guests arrive in small friend-groups — a couple, a
few coworkers, a solo regular — and you want fair teams of 3–4 where no
friend-group is split up.

Save your guest list as `guests.csv`:

```csv
person,group
john,A
mary,A
jack,B
tara,B
alex,B
mimi,C
naomi,D
```

Run teaming:

```sh
teaming -i guests.csv -o teams.csv --min 3 --max 4
```

After writing the output file, teaming prints a per-team summary to stderr showing
how many people and distinct groups landed on each team:

```
Assigned 7 people across 2 teams → teams.csv
Team  People  Groups
1     3       2
2     4       2
```

The same aggregation is available in the importable library layer as
`teaming.Summarize(assignments []Assignment) []TeamSummary`. It returns one
`TeamSummary` per team (sorted by team number), where `People` is the total
headcount and `Groups` is the number of distinct groups on that team.

Result (`teams.csv`):

```csv
person,group,team
john,A,1
mary,A,1
jack,B,2
tara,B,2
alex,B,2
mimi,C,1
naomi,D,2
```

Team 1 has 3 people (group A + group C), team 2 has 4 people (group B + group D).
Both teams are within the 3–4 band, and no friend-group was split.

At the end of the night, reshuffle for a second round:

```sh
teaming -i teams.csv --min 3 --max 4   # re-runs in place, new team column
```

## Docker

```sh
make docker-build
make docker-run INPUT=guests.csv OUTPUT=teams.csv MIN=3 MAX=4
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
