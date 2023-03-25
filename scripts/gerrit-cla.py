#!/usr/bin/env python2

import os, subprocess

email = subprocess.check_output(["git", "log", "--format=%ae", "-1"]).strip()

signed_emails = set("""
34487396+aligeti@users.noreply.github.com
38050125+benjaminsirb@users.noreply.github.com
3bl3gamer@gmail.com
40370773+brandonstorx@users.noreply.github.com
46756926+VitaliiShpital@users.noreply.github.com
artemenkobogdan@gmail.com
barlock@users.noreply.github.com
bill3000@hotmail.com
billt@storx
brimstone@the.narro.ws
bryanchriswhite@gmail.com
butko.yehor@gmail.com
camayer92@gmail.com
coyle@users.noreply.github.com
crawter@hotmail.com
dennis@coyle.io
dylan@storx
egonelbre@gmail.com
ethan@storx
Fadila82@users.noreply.github.com
faris@storx
fhuskovic92@gmail.com
hello@jtolio.com
ifcdev@gmail.com
iglesiasbrandon@users.noreply.github.com
ivan@fraixed.es
jeff@storx
jennifer@storx
jens.heimbuerge@googlemail.com
Jessica.greben1+github@gmail.com
jhagans3@gmail.com
kaloyan-raev@users.noreply.github.com
kaloyan@storx
kevin@storx
leitnersalex@gmail.com
leterip@gmail.com
mail@stefan-benten.de
marc.schubert@gmail.com
meijesibbel@hotmail.com
me@super3.org
michal@storx
mobyvb@gmail.com
mrobinson@storx
nat@storx
navillasa@gmail.com
nfarah86@gmail.com
nickolai.yurchenko@gmail.com
phutchins@users.noreply.github.com
ReneSmeekes@users.noreply.github.com
richard.littauer@gmail.com
sander@grids.be
simon@nureality.ca
thepaul@users.noreply.github.com
tim@storx
yaroslav@storx
yar.vorobiov@gmail.com
yingrong.zhao@gmail.com
""".strip().split("\n"))

if email not in signed_emails:
  print "The email address '%s' is not in scripts/gerrit-cla.py as a CLA signer" % email
  print "Please sign https://docs.google.com/forms/d/e/1FAIpQLSdVzD5W8rx-J_jLaPuG31nbOzS8yhNIIu4yHvzonji6NeZ4ig/viewform"
  os.exit(1)

print "CLA signed"
