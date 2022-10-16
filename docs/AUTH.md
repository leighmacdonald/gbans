# Authentication
    
## Web Clients

User authentication uses a combination of OpenID Connect (steam login) and a fairly standard OAuth2-like token flow

    +--------+                                            +---------------+    +---------------+
    |        |--(A)------- Authorization Grant -----------+---------------+--->|               |
    |        |                                            |               |    |  Steam OIC    |
    |        |<-(B)------- Access Token ------------------+---------------+----|               |
    |        |               & Refresh Token              |               |    +---------------+ 
    |        |                                            |               |    
    |        |--(C)------- Access Token ----------------->|               |
    | Client |<-(D)------- Protected Resource ------------|    gbans      |
    |        |--(E)------- Access Token ----------------->|               |
    |        |<-(F)------- Invalid Token Error -----------|               |
    |        |--(G)------- Refresh Token ---------------->|               |
    |        |<-(H)------- Access Token ------------------|               |   
    +--------+             & Optional Refresh Token       +---------------+    

A) The client requests an access token by authenticating with the
authorization server and presenting an authorization grant.

B) The authorization server authenticates the client and validates
the authorization grant, and if valid, issues an access token
and a refresh token.

C) The client makes a protected resource request to the resource
server by presenting the access token.

D) The resource server validates the access token, and if valid,
serves the request.

## SRCDS Game Servers (gbans sourcemod plugin)

The game instances use a fairly similar method except they remove the openid provider in favour
of a more simplified static key (created upon server creation) used as the refresh token.

    +--------+                                  
    |        |<-(A)------- Load Static Token From Disk        
    |        |                                            +---------------+
    |        |--(B)------- Request Access Token --------->|               |
    |        |<-(C)------- Receive Access Token ----------|               |
    |        |--(D)------- Access Token ----------------->+               |
    | SRCDS  |<-(E)------- Protected Resource ------------|    gbans      |
    |        |--(E)------- Access Token ----------------->|               |
    |        |<-(F)------- Invalid Token Error -----------|               |
    |        |--(G)------- Refresh Token ---------------->|               |
    |        |<-(H)------- Access Token ------------------|               |   
    +--------+                                            +---------------+    

A) The server loads the static refresh token from disk, this is created upon server setup.

B) Request an access token to gbans using the static token as credentials.

C) Receive JWT access token that the server can use to authenticate further requests

D) The client makes a protected resource request to the resource
server by presenting the access token.

E) The resource server validates the access token, and if valid,
serves the request.

