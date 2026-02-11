# Palo Alto TSF Data Mapping Template

## Purpose
Use this template to extract a consistent inventory from Palo Alto Networks tech support bundles (TSF), regardless of filename or device-specific content.

## Output fields
- Hostname
- Model
- Serial number
- PAN-OS version
- Management IP
- Licenses and status (Advanced Threat Prevention, WildFire, URL Filtering, etc.)
- Management type (Panorama-managed or cloud-managed/standalone)
- Panorama IP(s) if Panorama-managed
- HA status (enabled/disabled)
- HA mode
- HA peer
- Interfaces with configuration
- Zones
- Routes from configuration and runtime
- Device type classification (Firewall vs Panorama)
- Panorama managed device inventory
- Panorama device-group to firewall mappings
- Panorama template-stack to firewall mappings
- Cloud Logging Service forwarding enabled/disabled

## Source discovery (generic)
Find inputs using patterns, not fixed names.

- CLI aggregate output:
  - `tmp/cli/techsupport_*.txt`
  - Fallback: any large CLI text containing sections beginning with `> show ...` / `> request ...`
- Local config XML:
  - `**/saved-configs/running-config.xml`
  - `**/saved-configs/techsupport-saved-currcfg.xml`
- Panorama-pushed config XML (if present):
  - `**/panorama_pushed/mergesp.xml`
  - Fallback: `**/panorama_pushed/*push*.xml`

## Parsing model
- Prefer runtime command output for current state.
- Prefer config XML for intended configuration.
- If Panorama-pushed config exists, treat it as higher fidelity for interfaces/zones/VR than minimal local running config.

## Extraction map

### 1) Hostname
- Runtime primary:
  - Section header: `> show system info`
  - Regex: `^hostname:\s*(.+)$`
- Config fallback XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/hostname`

### 2) Model
- Runtime primary:
  - Section: `> show system info`
  - Regex: `^model:\s*(.+)$`
- Runtime fallback:
  - Section: `> show high-availability all`
  - Regex (local block): `^\s*Model:\s*(.+)$`

### 3) Serial number
- Runtime primary:
  - Section: `> show system info`
  - Regex: `^serial:\s*(.+)$`
- Runtime fallback:
  - Section: `> show high-availability all`
  - Regex: `^\s*Serial:\s*(.+)$`

### 4) PAN-OS version
- Runtime primary:
  - Section: `> show system info`
  - Regex: `^sw-version:\s*(.+)$`
- Runtime fallback:
  - Section: `> show high-availability all`
  - Regex: `^\s*Build Release:\s*(.+)$`

### 5) Management IP
- Runtime primary:
  - Section: `> show system info`
  - Regex: `^ip-address:\s*(.+)$`
- Runtime fallback:
  - Section: `> show interface management`
  - Regex: `^Ip address:\s*(.+)$`
- Config fallback XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/ip-address`

### 6) Licenses and status
- Runtime primary:
  - Section: `> request license info`
  - Parse repeating blocks split by `^License entry:`
  - Per block parse:
    - Feature: `^Feature:\s*(.+)$`
    - Status: `^Expired\?:\s*(yes|no)\s*$`
    - Expiration date: `^Expires:\s*(.+)$`
    - Optional metadata: `Description`, `Issued`, `Authcode`
- Normalize status:
  - `Expired?: no` => active
  - `Expired?: yes` => expired

### 7) Management type (Panorama / cloud / standalone)
- Panorama-managed detection (config):
  - Check existence of either XPath:
    - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server`
    - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server-2`
  - If present => `panorama-managed`
- Cloud indicator (runtime supplemental):
  - Section: `> show system info`
  - Regex: `^cloud-mode:\s*(.+)$`
- If no Panorama settings and no explicit cloud management indicators => `standalone/undetermined`

### 8) Panorama IP(s)
- Config XPath(s):
  - Primary: `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server`
  - Secondary: `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server-2`
- Return as list of unique non-empty IP/FQDN values.

### 9) HA status (enabled/disabled)
- Config primary XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/high-availability/enabled`
  - `yes` => enabled, `no`/missing => disabled or not configured
- Runtime supplemental:
  - Section: `> show high-availability all`
  - Presence of valid HA group output indicates operational HA context

### 10) HA mode
- Config primary XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/high-availability/group/mode/*`
  - Child tag name indicates mode (for example `active-passive`, `active-active`)
- Runtime fallback:
  - Section: `> show high-availability all`
  - Regex: `^\s*Mode:\s*(.+)$`

### 11) HA peer
- Config primary XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/high-availability/group/peer-ip`
  - Optional backup: `.../peer-ip-backup`
- Runtime supplemental:
  - Section: `> show high-availability all`
  - Parse peer block for peer mgmt IP and HA link IPs

### 12) Interfaces with configuration
- Config source priority:
  1. Panorama merged config (if present)
  2. Local running config
- Base XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/network/interface`
- Common interface trees:
  - `ethernet/entry`
  - `aggregate-ethernet/entry`
  - `.../layer3/units/entry`
  - `loopback/units/entry`, `vlan/units/entry`, `tunnel/units/entry`
- Runtime supplemental:
  - Section: `> show interface all`
  - Parse physical and logical tables for operational state

### 13) Zones
- Config source priority:
  1. Panorama merged config
  2. Local running config
- XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/vsys/entry/zone/entry`
- For each zone parse:
  - Zone name (`@name`)
  - Zone type (`network/layer3`, `network/virtual-wire`, etc.)
  - Member interfaces (`member` entries)

### 14) Routes from configuration
- Config source priority:
  1. Panorama merged config
  2. Local running config
- Virtual router base XPath:
  - `/config/devices/entry[@name='localhost.localdomain']/network/virtual-router/entry`
- Static route XPath (when present):
  - `.../static-route/entry`
  - Parse destination, nexthop, interface, metric where available
- Dynamic routing config (when present):
  - `.../protocol/*` (BGP/OSPF/RIP/etc.)
- Note: some deployments have zero static routes and rely on dynamic routing only.

### 15) Routes from runtime
- Runtime sections:
  - `> show routing summary`
  - `> show routing route`
  - Optional alternate commands in other TSFs: `show advanced-routing route`
- Parse route table columns:
  - destination, nexthop, metric, flags, interface, next-AS

### 16) Device type classification (for TSF scrape reports)
- Runtime primary (CLI):
  - Section: `> show system info`
  - Fields:
    - `model` via `^model:\s*(.+)$`
    - `system-mode` via `^system-mode:\s*(.+)$` (often present on Panorama)
- Classification rule:
  - If `model == Panorama` OR `system-mode == management-only` => `Panorama`
  - Else => `Firewall`

### 17) TSF-level "managed by" classification
- Panorama-managed firewall detection (config):
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server`
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system/panorama/local-panorama/panorama-server-2`
  - If either exists and is non-empty => managed by Panorama
- Cloud hint (runtime supplemental):
  - Section: `> show system info`
  - `cloud-mode` via `^cloud-mode:\s*(.+)$`
- Standalone fallback:
  - No Panorama server fields + no cloud-managed indicator => standalone/unknown

### 18) License entry count (for summary tables)
- Runtime source:
  - Section: `> request license info`
- Count rule:
  - Count occurrences of lines matching `^Feature:\s+`
  - Optional full block parser remains in section 6

### 19) Panorama managed device inventory
- Panorama config source:
  - `/config/mgt-config/devices/entry/@name`
- Meaning:
  - Each `@name` is a managed device serial known to Panorama
- Note:
  - This is Panorama-only data; not expected on firewall TSFs

### 20) Panorama device-group to firewall mappings
- Panorama config source:
  - Device-group names:
    - `/config/devices/entry[@name='localhost.localdomain']/device-group/entry/@name`
  - Firewalls bound to each device-group:
    - `/config/devices/entry[@name='localhost.localdomain']/device-group/entry[@name='{DG_NAME}']/devices/entry/@name`
  - Device-group referenced templates:
    - `/config/devices/entry[@name='localhost.localdomain']/device-group/entry[@name='{DG_NAME}']/reference-templates/member`
- Output shape recommendation:
  - `device_group_name`
  - `firewall_serials[]`
  - `reference_templates[]`

### 21) Panorama template-stack to firewall mappings
- Panorama config source:
  - Template-stack names:
    - `/config/devices/entry[@name='localhost.localdomain']/template-stack/entry/@name`
  - Firewalls in each template-stack:
    - `/config/devices/entry[@name='localhost.localdomain']/template-stack/entry[@name='{STACK_NAME}']/devices/entry/@name`
  - Templates included in each stack:
    - `/config/devices/entry[@name='localhost.localdomain']/template-stack/entry[@name='{STACK_NAME}']/templates/member`
- Output shape recommendation:
  - `template_stack_name`
  - `firewall_serials[]`
  - `templates[]`

### 22) Panorama template inventory (optional but useful)
- Panorama config source:
  - `/config/devices/entry[@name='localhost.localdomain']/template/entry/@name`
- Optional enrichment:
  - Pull template-local config blocks for zones/interfaces under each template when needed

### 23) Cloud Logging Service forwarding (enabled/disabled + region)
- Primary XPath (local saved config):
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/setting/logging/logging-service-forwarding/enable`
  - Region:
    - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/setting/logging/logging-service-forwarding/logging-service-regions`
- Related setting (optional companion field):
  - `/config/devices/entry[@name='localhost.localdomain']/deviceconfig/setting/logging/enhanced-application-logging/enable`
- Important behavior across TSFs:
  - On standalone/local-managed firewalls, this is often present in `saved-configs/running-config.xml` and `techsupport-saved-currcfg.xml`.
  - On Panorama-managed firewalls, this may be absent in local saved-configs and only visible in template-resolved config:
    - `opt/pancfg/mgmt/tmp/panorama_pushed/mergesp.xml`
- Recommended extraction order:
  1. `saved-configs/techsupport-saved-currcfg.xml`
  2. `saved-configs/running-config.xml`
  3. `tmp/panorama_pushed/mergesp.xml`
  4. Runtime operational check (if present): `> request logging-service-forwarding status`
- Normalization:
  - `yes` => enabled
  - `no` => disabled
  - missing node => unknown (do not coerce to disabled)

## Normalization recommendations
- Trim whitespace and preserve case for display fields.
- Convert booleans to canonical values (`true/false` or `enabled/disabled`).
- Return empty lists instead of null where practical.
- Keep raw section snippets for audit/debug when parse confidence is low.

## Robust fallback order
For each field:
1. Runtime command section (if present)
2. Panorama merged config XML (if present)
3. Local running config XML
4. Mark as `not_found`

## Validation checks
- Serial from `show system info` should match serial seen in HA/local device blocks.
- Management IP from runtime should match config `deviceconfig/system/ip-address`.
- If Panorama IPs exist, management type should be Panorama-managed.
- If HA enabled in config, HA mode should be present.

## Optional dedupe key (cross-file history)
There is no guaranteed universal TSF bundle UUID. Use deterministic fingerprints:
- Primary: SHA-256 of original uploaded archive (preferred)
- Fallback: SHA-256 of canonical CLI file content (`tmp/cli/techsupport_*.txt`)
- Supplemental identity tuple: `(serial_number, hostname, show_clock_time or capture timestamp)`
