# Design Doc for Thoth's Collection website

Thoth's Collection is a website for organizing your collections. The specific products in mind for the MVP are Magic: the Gathering cards and sealed product. Ideally features will be added later to manage other collections, such as books and DVDs.

## General notes

Backend written in Go

Frontend as React SPA, written in Typescript.

To be deployed as a subdomain on a VPS somewhere.

Use generated code wherever practical
 - The backend should manage the database code using sqlx
 - The backend API should be declared using Protobuf.
	- Because we need to support a webapp, use gRPC-Gateway as a proxy.
	- Use the Protobuf files to generate the frontend's client library

Use formatting linters to ensure uniform code styles.

# Frontend

React SPA in Typescript.

Accounts can be created using direct email+password, or via OAuth providers.
	- Out of the box, only Google auth is supported.

Use bookmarkable URLs wherever reasonable.

Are there automatic linters like `go fmt` for Typescript?

# Backend

Database will be a mysql database hosted over the internet.

Use `go fmt` to format the code.

Data for the cards will be periodically loaded from Scryfall's bulk data APIs. (we'll need a client library for that, too)

## Data Model

1. Users
2. Collection
	- Name, description. Freetext fields for users to remember which collection is which.
	- Type. This will describe what kind of collection it is: Loose MtG cards, Sealed MtG product, a MtG Deck, Books, DVDs, etc.
	- Foreign key to Users table to denote the Owner. 
	- Collections and Items should only be viewable by the owner by default, unless the owner marks it as Public. It should always only be writable by the Owner.
3. Items
	- These are the child elements of a Collection. Different fields are relevant for different types of collections.
	- For Cards in a collection or deck, this should be sufficient fields to join in the data from the Scryfall dataset - use a foreign key + field describing foil treatment.
	- For Sealed Product, just a Set ID and Description may be sufficient. Ideally it should be enough to look up prices automatically somewhere. 
	- For Books and DVDs, a UPC field and Title.
	- Include a Notes field on all Items so users can write freetext notes like "borrowed this from Rich"
	- Include a Quantity field rather than allowing duplicates in the Collection. This needs to be a compound constraint for Cards to include the foil treatment field.
	- Use separate tables for the different types of Items, but they should all use a forien key reference to their owning Collection.

Use Go classes for the application to handle these data objects separate from the sqlc models. Write helper functions to convert between database -> application -> wire protocol models.


