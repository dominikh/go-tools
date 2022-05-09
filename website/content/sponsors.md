---
title: "Supporting Staticcheck's open source development"
linkTitle: "Sponsors"
menu:
  main:
    weight: 4
    pre: <i class='fas fa-heart'></i>
---

{{< blocks/section type="section" color="white"  >}}

# Supporting Staticcheck's open source development

Staticcheck is an open source project that is provided free of charge and without any expectations.
Nevertheless, working on it requires a considerable time investment.
Some very generous people choose to support its development
by donating money on [GitHub Sponsors](https://github.com/users/dominikh/sponsorship) or [Patreon](https://www.patreon.com/dominikh).
While this is in no way expected of them, it is tremendously appreciated.

In addition to these individuals, a number of companies also
decide to support Staticcheck through monetary means.
Their contributions to open source ensure the viability and future development of Staticcheck.
Their support, too, is greatly appreciated.

{{< sponsors.inline >}}
{{ with $sponsors :=  $.Site.Data.sponsors.sponsors }}
The companies supporting Staticcheck are, in alphabetical order:

<ul>
  {{ range $sponsor := sort $sponsors "name" "asc" }}
  {{ if $sponsor.enabled }}
  <li><a href="{{ $sponsor.url }}">{{ $sponsor.name }}</a></li>
  {{ end }}
  {{ end }}
</ul>
{{ end }}
{{< /sponsors.inline >}}

If your company would like to support Staticcheck, please check out [GitHub Sponsors](https://github.com/users/dominikh/sponsorship)
or [get in touch with me directly.](mailto:dominik@honnef.co)
For [$250 USD a month](https://github.com/users/dominikh/sponsorship?utf8=%E2%9C%93&tier_id=MDIyOk1hcmtldHBsYWNlTGlzdGluZ1BsYW4yNTAy&editing=false),
we will proudly display your logo on the project's homepage,
showing the world that you truly care about code quality and open source.

Finally, every single user, individual and company alike, is to be thanked for using Staticcheck, providing feedback, requesting features and in general caring about code quality.
Without its users, there would be no Staticcheck.

{{< /blocks/section >}}
