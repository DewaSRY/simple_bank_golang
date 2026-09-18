> **Status: implemented.** See `app/[locale]/(public)/logout/page.tsx` (min
> 300ms spinner, then redirect to `/login`), `feature/auth/components/session-guard.tsx`
> (proactive profile fetch to surface an expired session), and
> `lib/api/api-interceptor.ts`'s `handleUnauthorized()` (401 → `/logout`).
> Kept here as the original task prompt, not as living documentation.

# logout flow 


current application should fetch user profile 


# the update is 

- if there is an invalid or the server throw un authentication user user will get redirect to logout page


- logout page is an page where the authentication data stored will get remove 
- currently api have no logout endpoint
- on the logout page user will see a loading or circle progress animation, the animation will last at minimum 300 milisecond ontil the data remove 
