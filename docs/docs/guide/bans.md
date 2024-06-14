# Ban functionality

This page outlines the various methods players can be banned. Currently, it's only possible to appeal
a steam ban. The rest are considered effectively permanent.

## Steam Bans

The standard ban usecase, banning by steam ID. Note the 2 special options for `Include Friends` and `IP Evading Allowed`.

If include friends is enabled, then all the ban recipients friends will also be banned. Friends do not receive a entry
in the ban table however, and are automatically allowed back once the parents ban is completed. The list of friends is only
updated periodically, so you may have to wait several hours for the changes to take effect and existing banned friends get 
flushed.

IP evasion option allows users to connect from the same ip as a currently banned user. When not enabled, users
are automatically banned for `evasion` and their ban lengths are changed to permanent. If you want a banned users "brother"
to be able to play, then be sure to enable this.

This type of ban is the only one that allows `muting` players.

## CIDR bans

Similar to steam bans, except they also match against a [cidr](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) range.

For example, to ban a user and his entire network block, you could ban `50.60.70.0/24`. This would ban anyone connecting 
from `50.60.70.0 - 50.60.70.255`.

## ASN Bans

[ANS](https://en.wikipedia.org/wiki/Autonomous_system_(Internet)) bans are probably the least used option. They should 
be used with care as they can be too broad in what they ban if you are not careful. If a user connects with a range that
the banned ASN owns, the user will be kicked. 


## Steam Group Bans

Ban all members of a particular steam group from connecting.

You can find a groups GID by opening the memberslist directly. You can achieve this by appending `/memberslistxml/?xml=1`
to the url of the steam group. You should end up with something like:

    https://steamcommunity.com/groups/valve/memberslistxml/?xml=1

Inside here, at the top, you should see the group ID you can use for banning:

```xml
<groupID64>103582791429521412</groupID64>
```

Similar to the friends list bans, these are also updated periodically. Any changes may take a few hours to flush 
through the system. 

Valve does not like hitting the memberslist endpoint that often without getting rate limited. Currently there is no 
protections for this, so its advised to not add too many entries for now.  