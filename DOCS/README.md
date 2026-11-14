# UPSTONK documentation

Everything here was checked against the code, or by running it, on 2026-11-15. Where the code and older docs disagreed, the code won.

| Section | What it covers |
|---|---|
| [getting-started/](getting-started/README.md) | Prerequisites, running the API and the GUI, configuration, smoke-test scripts |
| [architecture/](architecture/README.md) | The discovery pipeline, search providers, eligibility model, ranking |
| [api/](api/README.md) | Endpoints, request and response reference, validation, errors |
| [data-sources/](data-sources/README.md) | Each external data source and its current status |
| [development/](development/README.md) | Repo layout, tests, the GUI contract fixtures, adding rules and providers |
| [development/known-gaps.md](development/known-gaps.md) | What the code doesn't do yet |
| [UBIQUITOUS_LANGUAGE.md](UBIQUITOUS_LANGUAGE.md) | Glossary: the domain terms to use, the aliases to avoid, and known ambiguities |
| [plans/](plans/) | Work plans (e.g. [gaps-remediation.md](plans/gaps-remediation.md)) |

**Reading order for someone new:** getting-started, then architecture, then known-gaps. Read api/ when you need to call the server directly.
