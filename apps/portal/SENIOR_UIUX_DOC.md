Act as a **Senior Frontend Engineer, UI/UX Designer, and Creative Web Designer with 10+ years of experience building high-end SaaS and product websites**.

I have an existing application. Your first task is to **learn and understand the application before making any changes**.

After understanding the product, create a **beautiful, modern, premium landing page** that introduces the application to new users and communicates its value through strong visual storytelling.

---

# 1. First Understand My Application

Before implementing anything, explore the existing application thoroughly.

Understand:

- What the application does
- Who the target users are
- The main problems it solves
- The key features
- The most valuable workflows
- The application's unique selling points
- Important entities/data
- The existing visual identity
- Existing colors
- Typography
- Components
- Icons
- Brand assets
- Existing design system
- Existing frontend architecture

Do not immediately start coding.

First build a mental model of the product.

If something is unclear from the code, infer carefully from the existing UI and terminology rather than inventing unrelated features.

---

# 2. Landing Page Goal

Create a landing page that answers these questions naturally:

1. **What is this product?**
2. **Who is it for?**
3. **What problem does it solve?**
4. **Why should I care?**
5. **How does it work?**
6. **What makes it different?**
7. **What can I accomplish with it?**
8. **What should I do next?**

The landing page should feel like a **story**, not simply a collection of feature cards.

The visitor should naturally progress from:

**Problem → Understanding → Solution → Features → Benefits → Proof → Action**

---

# 3. Storytelling Structure

Design the landing page with a strong narrative.

A possible structure:

### Hero

Create a powerful first impression.

Include:

- Strong headline
- Supporting description
- Primary CTA
- Secondary CTA if appropriate
- Beautiful product visualization
- Subtle motion
- Strong visual composition

The hero should communicate the product's value within a few seconds.

Avoid generic marketing phrases.

The headline should be based on what the application actually does.

---

### Problem

Introduce the problem users currently experience.

Make the visitor think:

> "Yes, this is exactly the problem I have."

Use visual storytelling rather than a large block of text.

---

### Solution

Transition naturally from the problem into the application's solution.

Show how the product changes the user's workflow.

Use:

- Product screenshots
- Animated UI
- Progressive reveals
- Scroll-based transitions
- Visual comparisons

---

### Key Features

Show the most important features.

Do not simply create a grid of 10+ feature cards.

Instead, prioritize the **3–5 most valuable capabilities**.

Each feature should answer:

> "How does this make the user's life better?"

Use the actual application's functionality as the source of truth.

---

### Product Showcase

Create large visual sections where the actual application becomes the hero.

For example:

- Large dashboard preview
- Workflow animation
- Interactive UI demonstration
- Before/after visualization
- Scroll-driven product walkthrough

The product itself should be the primary visual asset.

---

### Benefits

Translate technical features into user outcomes.

For example:

Instead of:

> "Real-time synchronization"

Prefer:

> "Everyone always works with the latest information."

Focus on outcomes rather than implementation details.

---

### Trust / Proof

If the existing application contains relevant information, use it.

Possible elements:

- Statistics
- Metrics
- Customer/user information
- Integrations
- Security information
- Testimonials

**Do not invent testimonials, customers, statistics, or claims.**

If there is no real information available, omit this section or create a neutral alternative.

---

### Final CTA

End the story with a strong call to action.

The visitor should clearly understand:

> "What should I do now?"

Make the final section visually memorable.

---

# 4. Micro-interactions

I want the landing page to feel **alive and interactive**, but not distracting.

Use purposeful micro-interactions for:

- Buttons
- Navigation
- Cards
- Images
- Links
- Feature sections
- Scroll transitions
- Hover states
- Cursor interactions where appropriate
- Form interactions

Animations should communicate:

- Hierarchy
- State changes
- Relationships
- Progression
- Focus

Avoid animation simply for decoration.

Animations should feel **smooth, subtle, and premium**.

---

# 5. Parallax & Scroll Storytelling

Make meaningful use of **parallax and scroll-based animation**.

I want sections to visually transform as the user scrolls.

Possible effects:

- Background elements moving at different speeds
- Product screenshots entering the viewport
- Text revealing progressively
- Images scaling subtly
- Cards transitioning between states
- Elements moving along different scroll speeds
- Sticky storytelling sections
- Horizontal scrolling sections
- Pinned product demonstrations
- Layered visual compositions

Use animation to reinforce the story.

Do not make the entire page move excessively.

The experience should feel closer to a **premium product website** than a typical template landing page.

---

# 6. Recommended Animation Libraries

Use established libraries instead of manually implementing complex animation systems.

Prefer:

### Framer Motion / Motion

Use for:

- Component animations
- Page transitions
- Hover interactions
- Entrance animations
- Layout animations
- Micro-interactions

If the project already uses Motion/Framer Motion, reuse it.

### GSAP + ScrollTrigger

Use when more advanced scroll-driven animations are needed.

Especially useful for:

- Parallax
- Pinned sections
- Scroll timelines
- Complex storytelling sequences
- Horizontal scroll experiences

### Lenis

Consider using Lenis for smooth scrolling if it fits the project.

Use it carefully and ensure accessibility is not negatively affected.

Do not install every library automatically.

**Choose the smallest set of libraries that can achieve the desired experience.**

---

# 7. Animation Performance

Animations must remain smooth.

Prioritize:

- transform
- opacity
- GPU-friendly properties
- requestAnimationFrame through established libraries
- avoiding expensive layout calculations

Avoid excessive:

- box-shadow animations
- filter animations
- layout-triggering animations
- DOM manipulation
- JavaScript scroll listeners when unnecessary

The landing page should remain performant on mobile devices.

---

# 8. Navigation / Mega Menu

I specifically want a **large navigation menu / mega menu**.

Create a premium desktop navigation experience.

The navigation should potentially contain categories such as:

- Product
- Features
- Solutions
- Resources
- Company
- Pricing
- Documentation

However, **use categories that actually make sense for my application**.

The mega menu should:

- Have strong visual hierarchy
- Be easy to scan
- Have clear grouping
- Support icons or small visual previews where appropriate
- Have subtle entrance/exit animations
- Have excellent hover states
- Be keyboard accessible
- Close appropriately when clicking outside
- Work correctly with keyboard navigation

Do not create unnecessary menu categories just to make the menu larger.

---

# 9. Mobile Navigation

The navigation must be completely redesigned for mobile.

Do not simply squeeze the desktop mega menu into a small screen.

Create an appropriate mobile navigation experience with:

- Mobile menu
- Accordion/submenus where appropriate
- Touch-friendly controls
- Clear hierarchy
- Smooth open/close animation
- Proper focus management

---

# 10. Visual Direction

The final design should feel:

**Premium + Modern + Elegant + Interactive + Professional**

Think of the quality level of modern product websites such as:

- Linear
- Vercel
- Stripe
- Raycast
- Notion
- Framer

Use these only as **design inspiration**, not something to copy.

Avoid:

- Generic SaaS templates
- Excessive gradients
- Excessive glassmorphism
- Too many floating cards
- Excessive animations
- Huge amounts of text
- Random decorative elements
- Unnecessary 3D effects

The design should have a strong visual identity based on the actual application.

---

# 11. Responsive Design

The landing page must be **mobile-first and fully responsive**.

Test the design conceptually at:

- 320px
- 375px
- 390px
- 430px
- 768px
- 1024px
- 1280px
- 1440px+

Pay particular attention to:

- Hero composition
- Typography scaling
- Navigation
- Mega menu
- Product screenshots
- Parallax effects
- Sticky sections
- Horizontal scrolling
- CTA buttons
- Text wrapping
- Animation performance

Animations should be reduced or simplified on smaller devices when appropriate.

Respect:

`prefers-reduced-motion`

Users who disable motion should still receive the complete story and functionality.

---

# 12. Reuse the Existing Application

Whenever possible, use the application's actual:

- Components
- Screenshots
- UI elements
- Icons
- Data
- Branding
- Colors
- Typography
- Product terminology

The landing page should feel like it belongs to the application.

It should not look like a completely unrelated marketing website.

---

# 13. Technical Requirements

Keep the implementation:

- Type-safe
- Maintainable
- Componentized
- Responsive
- Accessible
- Performant
- SEO-friendly

Do not rewrite unrelated parts of the application.

Do not break existing functionality.

Avoid creating unnecessary abstractions.

---

# 14. SEO

Implement appropriate:

- Page title
- Meta description
- Open Graph metadata
- Semantic HTML
- Proper heading hierarchy
- Image alt text
- Structured content

The landing page should be understandable by both users and search engines.

---

# 15. Important Rule About Content

**Do not invent product capabilities.**

The existing application is the source of truth.

If you need marketing copy, derive it from what the application actually does.

If you are unsure about a claim, use neutral wording or ask me.

---

# 16. Implementation Process

Follow this workflow:

### Phase 1 — Explore

Understand the existing application.

### Phase 2 — Analyze

Identify:

- Target audience
- Core problem
- Core value proposition
- Key features
- Existing visual language

### Phase 3 — Design

Create the landing-page story and section structure.

### Phase 4 — Implement

Build the landing page using the existing technology stack.

### Phase 5 — Motion

Add micro-interactions and scroll-based storytelling.

### Phase 6 — Responsive

Optimize the entire experience for mobile, tablet, and desktop.

### Phase 7 — Polish

Review spacing, typography, animation timing, accessibility, and visual consistency.

---

# Final Quality Standard

Before considering the task complete, ask yourself:

> If this landing page were shown to a potential customer, would they immediately understand what the product does?

> Does the page tell a compelling story instead of simply displaying features?

> Does scrolling feel intentional and rewarding?

> Do the animations enhance the story?

> Does the navigation feel premium?

> Does the page look excellent on mobile?

> Does the landing page feel like a real product with a strong identity rather than a generated template?

If the answer is no, continue improving it.

**Build something that feels polished enough to ship to real users.**
