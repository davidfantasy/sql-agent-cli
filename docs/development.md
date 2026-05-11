# Local Development

## Loop

1. Edit code.
2. Run `bash scripts/format.sh`.
3. Run `bash scripts/build.sh`.
4. Run `bash scripts/test.sh`.
5. If touching skill files, run `bash scripts/build-skill.sh`.

## Notes

- Supports MySQL and PostgreSQL.
- Integration tests rely on local Docker containers.
- `scripts/test.sh` is responsible for bringing test databases up and down.
