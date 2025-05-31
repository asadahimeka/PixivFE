// on network error when clicking on a link, navigate to the page anyway
addEventListener("htmx:sendError", function (event) {
  if (event.target.tagName == "A") {
    // if the server is down, this will show the browser's default "Unable to connect" page, which is familiar to the user
    document.location = event.detail.pathInfo.requestPath;
  }
});

// make 4xx 5xx responses swap in as well
addEventListener("htmx:beforeOnLoad", function (event) {
  // console.log("%o", event)
  event.detail.shouldSwap = true;
  event.detail.isError = false;
});

// function closeNavigationMenu() {
//   document.getElementById("sidebar-toggler").checked = false
// }
// // browser built-in navigation
// addEventListener("popstate", (event) => {
//   closeNavigationMenu()
// });
// // htmx triggered navigation
// addEventListener("htmx:pushedIntoHistory", (event) => {
//   closeNavigationMenu()
// });
