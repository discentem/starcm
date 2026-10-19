{% for section in sections -%}
[{{ section.name }}]
{% for key, value in section.settings -%}
{{ key }} = {{ value }}
{% endfor %}
{% endfor -%}
