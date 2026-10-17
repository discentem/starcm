Welcome to {{ hostname }}.
{% if maintenance -%}
Maintenance window: {{ maintenance }}
{% endif -%}
Admins: {{ admins | join(", ") }}
