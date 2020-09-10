# Cheropatilla http server

This repo contains the implementation of the http server for the Cheropatilla website.

### Installation

1. run `go get github.com/luisguve/cherosite` and `go get github.com/luisguve/cheroapi`. Then run `go install github.com/luisguve/cherosite/cmd/cherosite` and `go install github.com/luisguve/cheroapi/cmd/...`. The following binaries will be installed in your $GOBIN: `cherosite`, `userapi`, `general` and `contents`. On setup, all of these must be running.
1. You will need to write a .toml file in order to configure the site to get it working. See cherosite.toml at the project root for an example.
1. Follow the installation instructions for the gRPC services in the [cheroapi project](https://github.com/luisguve/cheroapi#Installation).

To run the web application, run `cherosite`, `userapi`, `general` and `contents`, then visit **localhost:8000** from your browser, create a couple users and start following users, creating posts, replying posts and saving/unsaving them.

### Application overview

##### Root: "/"

**If you're not logged in**, the login/signin page is rendered. You can create an account with an email, name, patillavatar, username, alias (if blank, name is used as alias), description and password. Email and username must be unique; username is alphanumeric and underscores are allowed.

*Note:* patillavatar (profile pic) is optional; if you don't send a picture, it picks a random one from the default pics specified in the field patillavatars in the toml file.

![login/signin](login.png)

**When you're logged in**, the dashboard page is rendered. Here you can see the recent activity of the users you're following, your own recent activity and the posts you've saved. All of these contents are loaded in a **random fashion**.

![dashboard](empty_dashboard.png)

##### Navigation bar

![navigation bar](navbar.png)

A couple of buttons are displayed in this area:

1. A link to the root, where the website logo is supposed to be.
![logo](navbar_logo.png)
1. A ***Recycle*** button. This button is the **main feature** of the whole website. The idea is that when you press it, it loads more contents in a **random fashion**, depending upon the select input aside it, and builds ***local pages*** from these contents. The navigation across these local pages will be done through **PREV** and **NEXT** buttons.
![recycle](navbar_recycle.png)
1. A link to */explore*.
![expore](navbar_explore.png)
1. Your notifications, a link to your profile page and a button to logout.
![user data](navbar_user.png)

##### Explore page: "/explore"

This page displays posts from every section registered in the `sections` array in the .toml file in a random fashion.

![explore](explore.png)

##### My profile page: "/myprofile"

In this page, you can view and update your basic information.

![myprofile](myprofile.png)

##### Other users' profile page: "/profile?username={username}"

In this page, you can view the basic information of othe user, along with the recent activity for that user.

![user profile](user_profile.png)

##### Section page: "/{section_id}"

This page displays posts from a given section and a form to create a post on that section. The section id must match the id of one of the sections specified in the `sections` array in the .toml file.

![section](section.png)

##### Post page: "/{section_id}/{post_id}"

This page displays the content of a given post and the comments associated to that post, at the bottom. Note that the comments are also loaded in a **random fashion**.

You can also reply other comments, but these are loaded sequentially in chronological order.

![post](post.png)

### Application API


