####################################################################################

ci/cd
when there is an merging to main

- first make new image to github repository
- then update the ec2 intace docker container
- make the docker container will run on scheduler, only run on 8pm to 12am

- fix the env problem

portal

- deployment the portal
- update poratal, to handle the bug on main funsioanality
- update to have more documentation

Make documentation

- make new portal documentation about the code style and deployment flow

Make ui library

- make ui library/or template to use on create new next portal

make me github action for

rule of make pr

- there is no direct push from local main to cloud

- onle branch can make pr to main is dev
- the branch want to make pr to dev should have patter "epic/\*"
