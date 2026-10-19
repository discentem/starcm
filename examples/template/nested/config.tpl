[server]
host = {{ server.host }}
port = {{ server.port }}
{% for user in users %}
[user.{{ user.name }}]
groups = {{ user.groups | join(", ") }}
{%- if user.shell %}
shell = {{ user.shell }}
{%- endif %}
{% endfor %}
[build]
date = {{ now.formatted }}
