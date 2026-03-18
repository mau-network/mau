# Schema.org Types for Social Applications

This document provides a comprehensive reference of [Schema.org](https://schema.org/) types commonly used in Mau social applications, with practical examples and usage patterns.

## Table of Contents
1. [Overview](#overview)
2. [Social Media Types](#social-media-types)
3. [Content Types](#content-types)
4. [Action Types](#action-types)
5. [Person and Organization](#person-and-organization)
6. [Media Objects](#media-objects)
7. [Events and Calendar](#events-and-calendar)
8. [Creative Works](#creative-works)
9. [Custom Extensions](#custom-extensions)
10. [Best Practices](#best-practices)

---

## Overview

### What is Schema.org?

[Schema.org](https://schema.org/) is a collaborative vocabulary maintained by Google, Microsoft, Yahoo, and Yandex. It provides a structured way to describe entities on the web using JSON-LD.

**Why Mau uses Schema.org:**

- **Shared vocabulary** - Universal understanding across applications
- **Well-documented** - Extensive documentation and examples
- **Extensible** - Add custom properties while maintaining compatibility
- **Web-compatible** - Content can be understood by search engines and other web tools
- **Multilingual** - Built-in support for multiple languages

### JSON-LD Basics

Every Mau file follows this structure:

```json
{
  "@context": "https://schema.org",
  "@type": "TypeName",
  "property1": "value1",
  "property2": "value2"
}
```

**Required fields:**
- `@context` - Almost always `"https://schema.org"`
- `@type` - Schema.org type (e.g., `SocialMediaPosting`, `Article`, `Person`)

**Common optional fields:**
- `@id` - Unique identifier (URL) for this item
- `name` - The name/title of the item
- `description` - A short description
- `dateCreated` / `datePublished` / `dateModified` - Timestamps
- `author` - Person or Organization who created it

---

## Social Media Types

### SocialMediaPosting

Status updates, tweets, short posts, microblog entries.

**Schema.org reference:** [SocialMediaPosting](https://schema.org/SocialMediaPosting)

**Basic example:**
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "@id": "/p2p/5d000b.../hello-world.json.pgp",
  "headline": "Hello, decentralized world!",
  "articleBody": "This is my first Mau post. Excited to be here!",
  "author": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b2f2c040a1675b49d7f0c7cb7dc36999d56"
  },
  "datePublished": "2026-03-18T01:00:00Z"
}
```

**Key properties:**
- `headline` - Short title or first line
- `articleBody` - Main text content
- `author` - Person who posted
- `datePublished` - When it was posted
- `sharedContent` - Link to content being shared (repost/retweet)
- `commentCount` - Number of comments
- `interactionStatistic` - Detailed interaction stats

**With mentions:**
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Check out @bob's new project!",
  "mentions": [
    {
      "@type": "Person",
      "name": "Bob",
      "identifier": "a1234567890abcdef1234567890abcdef12345678"
    }
  ],
  "datePublished": "2026-03-18T02:00:00Z"
}
```

**With hashtags:**
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Building P2P apps with #mau #decentralization",
  "keywords": ["mau", "decentralization", "p2p"],
  "datePublished": "2026-03-18T03:00:00Z"
}
```

**Use cases:**
- Twitter-style status updates
- Facebook posts
- Mastodon toots
- Any short-form social content

---

### Comment

Comments, replies, discussions on content.

**Schema.org reference:** [Comment](https://schema.org/Comment)

**Basic example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "@id": "/p2p/a12345.../comment-1.json.pgp",
  "text": "Great post! Looking forward to more content.",
  "author": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "a1234567890abcdef1234567890abcdef12345678"
  },
  "dateCreated": "2026-03-18T01:30:00Z",
  "parentItem": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/5d000b.../hello-world.json.pgp"
  }
}
```

**Key properties:**
- `text` - The comment content
- `parentItem` - What's being commented on (post, article, etc.)
- `author` - Who wrote the comment
- `dateCreated` - When it was posted
- `upvoteCount` / `downvoteCount` - Vote counts

**Nested replies (threaded comments):**
```json
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "text": "I agree with Bob!",
  "parentItem": {
    "@type": "Comment",
    "@id": "/p2p/a12345.../comment-1.json.pgp"
  },
  "dateCreated": "2026-03-18T01:45:00Z"
}
```

**Use cases:**
- Reddit-style threaded comments
- Blog comment sections
- Discussion forums
- Code review comments

---

### Message

Private messages, direct messages, chat messages.

**Schema.org reference:** [Message](https://schema.org/Message)

**Basic example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Message",
  "@id": "/p2p/5d000b.../msg-to-bob.json.pgp",
  "text": "Hey Bob, want to grab coffee tomorrow?",
  "sender": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b2f2c040a1675b49d7f0c7cb7dc36999d56"
  },
  "recipient": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "a1234567890abcdef1234567890abcdef12345678"
  },
  "dateSent": "2026-03-18T10:00:00Z"
}
```

**Key properties:**
- `text` - Message content
- `sender` - Who sent it
- `recipient` - Who receives it (can be array for group chats)
- `dateSent` - When sent
- `dateReceived` - When received
- `dateRead` - When read (read receipts)

**Group chat:**
```json
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Hey everyone! Meeting at 3 PM.",
  "sender": {
    "@type": "Person",
    "identifier": "5d000b..."
  },
  "recipient": [
    { "@type": "Person", "identifier": "a12345..." },
    { "@type": "Person", "identifier": "b67890..." },
    { "@type": "Person", "identifier": "c11111..." }
  ],
  "dateSent": "2026-03-18T14:00:00Z"
}
```

**With attachments:**
```json
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Here's the photo I mentioned",
  "messageAttachment": {
    "@type": "ImageObject",
    "contentUrl": "file:///home/user/.mau/5d000b.../photo-beach.jpg",
    "encodingFormat": "image/jpeg"
  },
  "dateSent": "2026-03-18T15:00:00Z"
}
```

**Use cases:**
- WhatsApp-style messaging
- Telegram direct messages
- Slack DMs
- End-to-end encrypted chat

---

## Content Types

### Article

Blog posts, long-form content, articles, essays.

**Schema.org reference:** [Article](https://schema.org/Article)

**Basic example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "@id": "/p2p/5d000b.../building-p2p-apps.json.pgp",
  "headline": "Building P2P Apps with Mau",
  "alternativeHeadline": "A Practical Guide to Decentralized Social Networks",
  "articleBody": "In this comprehensive guide, I'll walk you through...",
  "author": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b..."
  },
  "datePublished": "2026-03-18T00:00:00Z",
  "dateModified": "2026-03-18T12:00:00Z",
  "wordCount": 2500,
  "keywords": ["p2p", "mau", "decentralization", "tutorial"]
}
```

**Key properties:**
- `headline` - Main title
- `alternativeHeadline` - Subtitle
- `articleBody` - Full text content
- `author` - Writer
- `datePublished` - First published
- `dateModified` - Last edited
- `wordCount` - Length in words
- `keywords` - Tags/categories
- `image` - Featured image

**With section structure:**
```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Complete Mau Tutorial",
  "articleBody": "...",
  "articleSection": "Technology",
  "hasPart": [
    {
      "@type": "WebPageElement",
      "headline": "Introduction",
      "text": "Mau is a p2p convention..."
    },
    {
      "@type": "WebPageElement",
      "headline": "Getting Started",
      "text": "First, install the CLI..."
    }
  ],
  "datePublished": "2026-03-18T00:00:00Z"
}
```

**Use cases:**
- Blog posts
- News articles
- Technical documentation
- Long-form essays
- Tutorials

---

### BlogPosting

More specific type for blog entries (subtype of Article).

**Schema.org reference:** [BlogPosting](https://schema.org/BlogPosting)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "March 2026 Update",
  "articleBody": "This month I've been working on...",
  "author": {
    "@type": "Person",
    "name": "Alice"
  },
  "datePublished": "2026-03-01T00:00:00Z",
  "blogPost": true
}
```

**Use for:** Traditional blog posts, personal journals, dev logs.

---

### Note

Short notes, reminders, snippets (simpler than Article).

**Schema.org reference:** [Note](https://schema.org/Note) (under CreativeWork)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Note",
  "text": "Remember to follow up with Bob about the project",
  "author": {
    "@type": "Person",
    "identifier": "5d000b..."
  },
  "dateCreated": "2026-03-18T09:00:00Z"
}
```

**Use cases:**
- Personal notes
- Todo items
- Reminders
- Quick snippets

---

### Recipe

Cooking recipes, instructions, ingredients.

**Schema.org reference:** [Recipe](https://schema.org/Recipe)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "@id": "/p2p/5d000b.../pasta-carbonara.json.pgp",
  "name": "Authentic Pasta Carbonara",
  "author": {
    "@type": "Person",
    "name": "Alice"
  },
  "datePublished": "2026-03-15T00:00:00Z",
  "image": {
    "@type": "ImageObject",
    "contentUrl": "file:///home/user/.mau/photos/carbonara.jpg"
  },
  "description": "Traditional Roman pasta dish with eggs, guanciale, and pecorino",
  "prepTime": "PT10M",
  "cookTime": "PT20M",
  "totalTime": "PT30M",
  "recipeYield": "4 servings",
  "recipeCategory": "Main Course",
  "recipeCuisine": "Italian",
  "recipeIngredient": [
    "400g spaghetti",
    "200g guanciale (or pancetta)",
    "4 large eggs",
    "100g Pecorino Romano cheese, grated",
    "Black pepper to taste",
    "Salt for pasta water"
  ],
  "recipeInstructions": [
    {
      "@type": "HowToStep",
      "text": "Bring a large pot of salted water to boil. Cook spaghetti according to package directions until al dente."
    },
    {
      "@type": "HowToStep",
      "text": "Cut guanciale into small cubes. Fry in a large pan over medium heat until crispy, about 5-7 minutes."
    },
    {
      "@type": "HowToStep",
      "text": "In a bowl, whisk eggs with grated Pecorino and plenty of black pepper."
    },
    {
      "@type": "HowToStep",
      "text": "Reserve 1 cup of pasta water, then drain pasta. Add pasta to the pan with guanciale (off heat)."
    },
    {
      "@type": "HowToStep",
      "text": "Pour egg mixture over pasta, tossing quickly. Add pasta water gradually to create a creamy sauce. The heat from pasta should cook the eggs without scrambling."
    },
    {
      "@type": "HowToStep",
      "text": "Serve immediately with extra Pecorino and black pepper."
    }
  ],
  "nutrition": {
    "@type": "NutritionInformation",
    "calories": "650 calories",
    "proteinContent": "28g",
    "fatContent": "35g"
  },
  "aggregateRating": {
    "@type": "AggregateRating",
    "ratingValue": "4.8",
    "reviewCount": "127"
  }
}
```

**Use cases:**
- Recipe sharing apps
- Cooking blogs
- Food social networks
- Community cookbooks

---

## Action Types

Actions represent interactions users perform on content or with other users.

### LikeAction

Likes, favorites, upvotes.

**Schema.org reference:** [LikeAction](https://schema.org/LikeAction)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "LikeAction",
  "@id": "/p2p/a12345.../like-alice-post.json.pgp",
  "agent": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "a1234567..."
  },
  "object": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/5d000b.../hello-world.json.pgp"
  },
  "startTime": "2026-03-18T01:15:00Z"
}
```

**Key properties:**
- `agent` - Who performed the action (Person)
- `object` - What was liked (any content type)
- `startTime` - When the like happened
- `endTime` - When unlike happened (optional)

**Use cases:**
- Facebook likes
- Twitter hearts
- Reddit upvotes
- Instagram likes

---

### FollowAction

Following users, subscribing to content.

**Schema.org reference:** [FollowAction](https://schema.org/FollowAction)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "FollowAction",
  "@id": "/p2p/a12345.../follow-alice.json.pgp",
  "agent": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "a12345..."
  },
  "object": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b..."
  },
  "startTime": "2026-03-18T10:00:00Z"
}
```

**Use cases:**
- Twitter following
- Instagram following
- RSS subscription equivalents
- Friend connections

---

### ShareAction

Sharing, reposting, retweeting content.

**Schema.org reference:** [ShareAction](https://schema.org/ShareAction)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "ShareAction",
  "@id": "/p2p/a12345.../share-article.json.pgp",
  "agent": {
    "@type": "Person",
    "identifier": "a12345..."
  },
  "object": {
    "@type": "Article",
    "@id": "/p2p/5d000b.../building-p2p-apps.json.pgp"
  },
  "startTime": "2026-03-18T15:00:00Z",
  "description": "This is an excellent tutorial!"
}
```

**Use cases:**
- Twitter retweets
- Facebook shares
- Reddit crossposts
- Mastodon boosts

---

### ReviewAction

Reviews, ratings, evaluations.

**Schema.org reference:** [ReviewAction](https://schema.org/ReviewAction) and [Review](https://schema.org/Review)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Review",
  "@id": "/p2p/a12345.../review-recipe.json.pgp",
  "reviewBody": "Tried this recipe and it was amazing! Family loved it.",
  "reviewRating": {
    "@type": "Rating",
    "ratingValue": 5,
    "bestRating": 5
  },
  "author": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "a12345..."
  },
  "itemReviewed": {
    "@type": "Recipe",
    "@id": "/p2p/5d000b.../pasta-carbonara.json.pgp"
  },
  "datePublished": "2026-03-18T20:00:00Z"
}
```

**Use cases:**
- Product reviews
- Movie/book reviews
- Recipe ratings
- Service reviews

---

### ReplyAction

Formal reply to content (alternative to Comment).

**Schema.org reference:** [ReplyAction](https://schema.org/ReplyAction)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "ReplyAction",
  "agent": {
    "@type": "Person",
    "identifier": "a12345..."
  },
  "object": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/5d000b.../post.json.pgp"
  },
  "result": {
    "@type": "Comment",
    "@id": "/p2p/a12345.../reply.json.pgp",
    "text": "Great points!"
  },
  "startTime": "2026-03-18T16:00:00Z"
}
```

---

## Person and Organization

### Person

Represents a user, author, contact.

**Schema.org reference:** [Person](https://schema.org/Person)

**Profile example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Person",
  "@id": "/p2p/5d000b.../profile.json.pgp",
  "identifier": "5d000b2f2c040a1675b49d7f0c7cb7dc36999d56",
  "name": "Alice Smith",
  "givenName": "Alice",
  "familyName": "Smith",
  "email": "alice@example.com",
  "url": "https://alice.example.com",
  "image": {
    "@type": "ImageObject",
    "contentUrl": "file:///home/alice/.mau/avatar.jpg"
  },
  "description": "Software engineer, open source enthusiast, coffee lover",
  "jobTitle": "Senior Developer",
  "worksFor": {
    "@type": "Organization",
    "name": "Acme Corp"
  },
  "address": {
    "@type": "PostalAddress",
    "addressLocality": "San Francisco",
    "addressRegion": "CA",
    "addressCountry": "US"
  },
  "birthDate": "1990-05-15",
  "sameAs": [
    "https://twitter.com/alice",
    "https://github.com/alice",
    "https://linkedin.com/in/alice"
  ]
}
```

**Key properties:**
- `identifier` - PGP fingerprint (primary key in Mau)
- `name` - Full display name
- `email` - Contact email
- `image` - Profile picture
- `description` - Bio
- `sameAs` - Links to other profiles (social media)
- `url` - Personal website

**Minimal Person (in content):**
```json
{
  "@type": "Person",
  "name": "Alice",
  "identifier": "5d000b..."
}
```

---

### Organization

Companies, projects, groups.

**Schema.org reference:** [Organization](https://schema.org/Organization)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Organization",
  "name": "Mau Network",
  "url": "https://mau-network.org",
  "logo": {
    "@type": "ImageObject",
    "contentUrl": "https://mau-network.org/logo.png"
  },
  "description": "Decentralized social networking protocol",
  "foundingDate": "2024",
  "founder": {
    "@type": "Person",
    "name": "Emad Elsaid"
  },
  "sameAs": [
    "https://github.com/mau-network"
  ]
}
```

---

## Media Objects

### ImageObject

Photos, images, graphics.

**Schema.org reference:** [ImageObject](https://schema.org/ImageObject)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "ImageObject",
  "@id": "/p2p/5d000b.../photo-sunset.json.pgp",
  "name": "Sunset at the Beach",
  "description": "Beautiful sunset I captured last weekend",
  "contentUrl": "file:///home/user/.mau/5d000b.../sunset.jpg",
  "encodingFormat": "image/jpeg",
  "width": "4000",
  "height": "3000",
  "fileSize": "2.5MB",
  "creator": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b..."
  },
  "dateCreated": "2026-03-15T18:30:00Z",
  "locationCreated": {
    "@type": "Place",
    "name": "Santa Cruz Beach"
  },
  "exifData": {
    "@type": "PropertyValue",
    "name": "Camera",
    "value": "Canon EOS R5"
  }
}
```

**Key properties:**
- `contentUrl` - Location of the image file
- `encodingFormat` - MIME type (image/jpeg, image/png, etc.)
- `width` / `height` - Dimensions in pixels
- `caption` - Image caption
- `thumbnail` - Smaller preview version
- `exifData` - Camera metadata

---

### VideoObject

Videos, movies, clips.

**Schema.org reference:** [VideoObject](https://schema.org/VideoObject)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "VideoObject",
  "@id": "/p2p/5d000b.../tutorial-video.json.pgp",
  "name": "Mau Tutorial - Getting Started",
  "description": "Learn how to build your first Mau application",
  "contentUrl": "file:///home/user/.mau/5d000b.../tutorial.mp4",
  "thumbnailUrl": "file:///home/user/.mau/5d000b.../tutorial-thumb.jpg",
  "encodingFormat": "video/mp4",
  "duration": "PT15M30S",
  "width": "1920",
  "height": "1080",
  "uploadDate": "2026-03-18T00:00:00Z",
  "creator": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b..."
  }
}
```

**Duration format:** ISO 8601 duration (PT15M30S = 15 minutes 30 seconds)

---

### AudioObject

Music, podcasts, audio recordings.

**Schema.org reference:** [AudioObject](https://schema.org/AudioObject)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "AudioObject",
  "name": "Podcast Episode 42: Decentralization",
  "description": "Discussion about P2P technologies and the future of social media",
  "contentUrl": "file:///home/user/.mau/5d000b.../podcast-ep42.mp3",
  "encodingFormat": "audio/mpeg",
  "duration": "PT1H23M45S",
  "datePublished": "2026-03-18T00:00:00Z"
}
```

---

### MediaObject

Generic media (when type is unclear or mixed).

**Schema.org reference:** [MediaObject](https://schema.org/MediaObject)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "MediaObject",
  "name": "Presentation Slides",
  "contentUrl": "file:///home/user/.mau/5d000b.../slides.pdf",
  "encodingFormat": "application/pdf"
}
```

---

## Events and Calendar

### Event

Meetings, conferences, gatherings, appointments.

**Schema.org reference:** [Event](https://schema.org/Event)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Event",
  "@id": "/p2p/5d000b.../mau-meetup.json.pgp",
  "name": "Mau Developer Meetup",
  "description": "Monthly gathering for Mau developers to discuss progress and ideas",
  "startDate": "2026-03-25T18:00:00Z",
  "endDate": "2026-03-25T20:00:00Z",
  "location": {
    "@type": "Place",
    "name": "Tech Hub",
    "address": {
      "@type": "PostalAddress",
      "streetAddress": "123 Main St",
      "addressLocality": "San Francisco",
      "addressRegion": "CA",
      "postalCode": "94102",
      "addressCountry": "US"
    }
  },
  "organizer": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b..."
  },
  "attendee": [
    { "@type": "Person", "name": "Bob", "identifier": "a12345..." },
    { "@type": "Person", "name": "Carol", "identifier": "c67890..." }
  ],
  "eventStatus": "https://schema.org/EventScheduled"
}
```

**Virtual event:**
```json
{
  "@context": "https://schema.org",
  "@type": "Event",
  "name": "Online Webinar: Building with Mau",
  "startDate": "2026-03-20T15:00:00Z",
  "endDate": "2026-03-20T16:00:00Z",
  "location": {
    "@type": "VirtualLocation",
    "url": "https://meet.example.com/mau-webinar"
  },
  "eventAttendanceMode": "https://schema.org/OnlineEventAttendanceMode"
}
```

---

## Creative Works

### Book

Books, eBooks, published works.

**Schema.org reference:** [Book](https://schema.org/Book)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "Book",
  "name": "The Decentralized Web",
  "author": {
    "@type": "Person",
    "name": "Alice Smith"
  },
  "isbn": "978-1234567890",
  "datePublished": "2025-01-15",
  "numberOfPages": 320,
  "bookFormat": "https://schema.org/EBook",
  "inLanguage": "en",
  "publisher": {
    "@type": "Organization",
    "name": "Tech Press"
  },
  "description": "A comprehensive guide to building decentralized applications"
}
```

---

### MusicRecording

Songs, tracks, albums.

**Schema.org reference:** [MusicRecording](https://schema.org/MusicRecording)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "MusicRecording",
  "name": "Digital Freedom",
  "byArtist": {
    "@type": "Person",
    "name": "The P2P Band"
  },
  "inAlbum": {
    "@type": "MusicAlbum",
    "name": "Decentralized Dreams"
  },
  "duration": "PT3M45S",
  "datePublished": "2026-01-01"
}
```

---

### SoftwareApplication

Software, apps, programs.

**Schema.org reference:** [SoftwareApplication](https://schema.org/SoftwareApplication)

**Example:**
```json
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "Mau Desktop",
  "applicationCategory": "SocialNetworkingApplication",
  "operatingSystem": "Linux, macOS, Windows",
  "softwareVersion": "1.0.0",
  "datePublished": "2026-03-01",
  "author": {
    "@type": "Person",
    "name": "Alice"
  },
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD"
  }
}
```

---

## Custom Extensions

### Extending Schema.org

You can add custom properties while maintaining Schema.org compatibility:

**Method 1: Custom context**
```json
{
  "@context": [
    "https://schema.org",
    {
      "mau": "https://mau-network.org/vocab/",
      "replyCount": "mau:replyCount",
      "visibility": "mau:visibility",
      "encryptedFor": "mau:encryptedFor"
    }
  ],
  "@type": "SocialMediaPosting",
  "headline": "Custom properties example",
  "mau:replyCount": 15,
  "mau:visibility": "friends-only",
  "mau:encryptedFor": ["5d000b...", "a12345..."]
}
```

**Method 2: additionalProperty**
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Post with metadata",
  "additionalProperty": [
    {
      "@type": "PropertyValue",
      "name": "clientApp",
      "value": "Mau Desktop 1.0"
    },
    {
      "@type": "PropertyValue",
      "name": "contentWarning",
      "value": "spoilers"
    }
  ]
}
```

**Common custom properties for Mau:**
- `visibility` - public / friends / private / custom
- `encryptedFor` - List of fingerprints
- `replyCount` / `likeCount` / `shareCount` - Interaction stats
- `contentWarning` - CW/trigger warnings
- `pinned` - Whether post is pinned
- `editHistory` - Links to previous versions

---

## Best Practices

### 1. Always Include @context and @type

**Good:**
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello world"
}
```

**Bad:**
```json
{
  "headline": "Hello world"
}
```

---

### 2. Use Appropriate Types

Choose the most specific type that fits your content:

- Status update? Use `SocialMediaPosting`, not `Article`
- Long blog post? Use `Article` or `BlogPosting`, not `SocialMediaPosting`
- Private message? Use `Message`, not `Comment`

---

### 3. Include Timestamps

Always include creation/publication dates:

```json
{
  "@type": "Article",
  "headline": "My Post",
  "datePublished": "2026-03-18T00:00:00Z",
  "dateModified": "2026-03-18T12:00:00Z"
}
```

**Use ISO 8601 format:** `YYYY-MM-DDTHH:MM:SSZ`

---

### 4. Reference Other Content with @id

Link to other files using full paths:

```json
{
  "@type": "Comment",
  "text": "Great post!",
  "parentItem": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/5d000b.../post.json.pgp"
  }
}
```

Don't forget the `.pgp` extension in the `@id`!

---

### 5. Keep Person Objects Consistent

Always include the fingerprint as `identifier`:

```json
{
  "@type": "Person",
  "name": "Alice",
  "identifier": "5d000b2f2c040a1675b49d7f0c7cb7dc36999d56"
}
```

This allows clients to verify signatures and link content to the correct user.

---

### 6. Use Standard Vocabularies When Possible

Before creating custom properties, check if Schema.org already has what you need:

- `keywords` instead of `tags`
- `dateModified` instead of `updatedAt`
- `author` instead of `creator` (for social content)

---

### 7. Support Multilingual Content

Use `@language` for internationalization:

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": {
    "@value": "Hello World",
    "@language": "en"
  },
  "alternativeHeadline": [
    { "@value": "مرحبا بالعالم", "@language": "ar" },
    { "@value": "Hola Mundo", "@language": "es" }
  ]
}
```

---

### 8. Validate Your JSON-LD

Test your content:
- [JSON-LD Playground](https://json-ld.org/playground/)
- [Schema.org Validator](https://validator.schema.org/)
- [Google Rich Results Test](https://search.google.com/test/rich-results)

---

### 9. Document Custom Extensions

If you create custom properties, document them:

```markdown
## Custom Mau Properties

- `mau:visibility` - String: "public" | "friends" | "private"
- `mau:encryptedFor` - Array of strings: PGP fingerprints
- `mau:replyCount` - Integer: Number of replies
```

Make your extension discoverable by publishing a vocabulary document.

---

### 10. Future-Proof Your Content

Schema.org evolves. Clients should:
- Ignore unknown properties (forward compatibility)
- Provide sensible defaults for missing properties
- Validate required fields only

Write content that works even if schema changes.

---

## Additional Resources

### Official Documentation
- [Schema.org Full Hierarchy](https://schema.org/docs/full.html)
- [JSON-LD Specification](https://www.w3.org/TR/json-ld11/)
- [Schema.org Extension Mechanism](https://schema.org/docs/extension.html)

### Tools
- [JSON-LD Playground](https://json-ld.org/playground/) - Test and visualize
- [Schema.org Search](https://schema.org/docs/search_results.html) - Find types
- [Google Structured Data Testing Tool](https://search.google.com/structured-data/testing-tool)

### Related Mau Documentation
- [Storage and Data Format (04-storage-and-data.md)](04-storage-and-data.md) - How files are structured
- [Building Social Apps (08-building-social-apps.md)](08-building-social-apps.md) - Practical patterns
- [HTTP API Reference (07-http-api.md)](07-http-api.md) - API endpoints

---

## Next Steps

Now that you understand Schema.org types:

1. **Build content** - Create posts, articles, and messages
2. **Define your app's types** - Choose which types your application uses
3. **Extend when needed** - Add custom properties for your use case
4. **Validate** - Test your JSON-LD is correct

For implementation guidance, see [API Reference (11-api-reference.md)](11-api-reference.md).

---

*Questions? Check [Troubleshooting (13-troubleshooting.md)](13-troubleshooting.md) or open an issue on [GitHub](https://github.com/mau-network/mau/issues).*
