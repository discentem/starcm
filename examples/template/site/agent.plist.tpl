<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{ label }}</string>
    <key>ProgramArguments</key>
    <array>
{%- for arg in program %}
        <string>{{ arg }}</string>
{%- endfor %}
    </array>
    <key>StartInterval</key>
    <integer>{{ interval_seconds }}</integer>
{%- if run_at_load %}
    <key>RunAtLoad</key>
    <true/>
{%- endif %}
</dict>
</plist>
