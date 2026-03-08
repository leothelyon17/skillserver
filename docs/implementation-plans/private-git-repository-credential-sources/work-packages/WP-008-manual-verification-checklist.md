# WP-008 Manual Verification Checklist

## Scope
Manual checks for behaviors that remain hard to automate end-to-end:
- UI masking around stored credentials and sync errors.
- SSH `known_hosts` handling against a real host key.

## Preconditions
- Run SkillServer with private git support enabled.
- Ensure at least one reachable private test repository exists for HTTPS and SSH flows.
- For stored-credential tests, configure:
  - `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=true`
  - one master-key source (`SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` or `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`)
  - persistence enabled (`SKILLSERVER_PERSISTENCE_DATA=true`)

## UI Masking Checklist
1. Add a private repo using `auth.mode=https_token` and `source=stored` with a unique token value.
   Expected: add succeeds and UI shows configured state without displaying token plaintext.
2. Refresh the page and reopen the same repo in edit mode.
   Expected: secret fields are empty/masked placeholders; prior secret value is not present in DOM text inputs.
3. Trigger a sync failure intentionally (temporary bad token) and run manual sync.
   Expected: `last_sync_error` appears in UI, but secret values and raw file paths are redacted.
4. Switch the repo back to a valid token and sync again.
   Expected: sync recovers to success; no historical secret value appears in response payloads or UI.

## SSH Known Hosts Checklist
1. Add a private repo using `auth.mode=ssh_key`, `source=file`, and provide refs for key, optional passphrase, and `known_hosts`.
   Expected: save succeeds only when `known_hosts_ref` is provided.
2. Run sync with matching host key entry in `known_hosts`.
   Expected: clone/pull succeeds.
3. Replace `known_hosts` with mismatched or malformed host key data and run sync.
   Expected: sync fails with host-verification error; no private key/passphrase content appears in API/UI errors.
4. Restore valid `known_hosts` and sync again.
   Expected: sync succeeds after correction without restarting the server.

## Regression Spot Checks
1. Add a public repo via URL-only payload and run sync/toggle/delete.
   Expected: all operations remain functional with `auth_mode=none`, `credential_source=none`, and `has_credentials=false`.
2. Rotate env/file-backed credential refs and rerun sync.
   Expected: subsequent sync attempts use refreshed values without process restart.
