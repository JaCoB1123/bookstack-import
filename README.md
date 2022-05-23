# bookstack-import

Imports a directory of markdown files to Bookstack

This was created to allow an automated migration from a folder of Markdown-Documents to BookStack. In our case the source-files were created by the excellent [theohbrothers/ConvertOneNote2MarkDown](https://github.com/theohbrothers/ConvertOneNote2MarkDown) from a OneNote notebook.

## Running a migration

To run a migration the following environment variables are needed:

```bash
export BOOKSTACK_URL=http://127.0.0.1
export BOOKSTACK_TOKEN_ID=
export BOOKSTACK_TOKEN_SECRET=
export BOOKSTACK_IMPORT_PATH=./

./bookstack-import
```