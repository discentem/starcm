dots: {{ user.name }} lives in {{ user.address.city }}
brackets: {{ user["name"] }} {{ user.pets[1]["name"] }}
list: {% for p in user.pets %}{{ p.kind }}={{ p.name }} {% endfor %}
dict: {% for k, v in env %}{{ k }}={{ v }} {% endfor %}
tuple: {{ point[0] }},{{ point[1] }}
none: [{{ nothing }}] {% if nothing %}set{% else %}unset{% endif %}
struct: {{ greeting.value }} success={{ greeting.success }}
