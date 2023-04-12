#! /bin/bash
cros-repo debug --build --boards brya,corsola,cherry,hatch --packages adhd
cros-repo common --sync --build --boards brya,corsola,cherry,hatch --packages adhd
cros-repo stable --build --boards brya,corsola,cherry,hatch --packages adhd