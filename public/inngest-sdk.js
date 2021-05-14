(function() {

  var CACHE_KEY = "inngest_user";
  var VERSION = "0.1.1";

  var defaults = {
    host: "inn.gs",
  };

  // store the key locally so as not to expose it to other JS code.
  var key = "";

  // user stores user data after a call to identify().  this is mixed
  // in with events with calls to track().
  var user = {};

  var Inngest = function () {};
  globalThis.Inngest = Inngest;

  Inngest.options = defaults;

  /**
   * init initializes Inngest given an ingest key.  Any options provided override
   * the defaults above.
   *
   */
  Inngest.init = function(k, options) {
    var self = this;
    key = k;

    assign(self.options, options || {});
    user = get(CACHE_KEY) || {};
  }

  Inngest.event = function(event) {
    var errors = validate(event);
    if (errors.length > 0) {
      console.warn("inngest event is invalid: ", errors.join(", "));
      return false;
    }
    event.data = event.data || {};
    assign(event.data, context());

    // The event.user object should take precedence over the identify() attributes
    // called.  Copy the event user attributes into a new variable so that we can
    // merge this into `user` from identify, then replace the original event.user.
    var overwritten = {};
    assign(overwritten, user);
    assign(overwritten, event.user || {});
    event.user = overwritten;

    var body = JSON.stringify(event);
    var url = "https://" + this.options.host + "/e/" + key;
    req(url, { "content-type": "application/json" }, body);
    return true;
  }

  Inngest.track = function(eventName, data) {
    evt = {
      name: eventName,
      data: assign(data || {}, context()),
      user: user,
    }
    Inngest.event(evt);
  }

  Inngest.identify = function(userID, data) {
    var map = {}; 
    if (typeof userID === "string") {
      map.external_id = userID;
    }
    if (typeof userID === "object") {
      data = userID;
    }

    // TODO: Should we merge previous identify values?
    assign(map, data);
    set(CACHE_KEY, map);

    // update local memory so that we don't have to wait on localStorage
    // and JSON serialization for each call.
    user = map;

    // XXX: Should we send an "identified" event here to _guarantee_ that the
    // contact is upserted?
  }

  function set(key, val) {
    try {
      window.localStorage.setItem(key, JSON.stringify(val));
    } catch(e) {
      console.warn(e);
    }
  }

  function get(key) {
    try {
      return JSON.parse(window.localStorage.getItem(key) || "null");
    } catch(e) {
      return null;
    }
  }

  function assign(to, from) {
    iter(from, function(key, val) {
      to[key] = val;
    });
  }

  function iter(obj, callback) {
    if (typeof obj !== "object" || obj === null || obj === undefined) {
      return;
    }
    for (var o in obj) {
      if (obj.hasOwnProperty(o)) {
        callback(o, obj[o]);
      }
    }
  }

  function validate(event) {
    var errors = [];

    if (!key) {
      errors.push("init() has not been called with an ingest key");
    }

    if (!event.name) {
      errors.push("event must have a name");
    }
    return errors;
  }

  function context() {
    return {
      context_path: window.location.pathname,
      context_url: window && window.location.href,
      context_title: document && document.title,
      context_search: window && window.location.search,
      context_referrer: document && document.referrer,
      context_user_agent: navigator && navigator.userAgent,
      context_library: "js",
      context_library_version: VERSION,
    };
  }

  function req(url, headers, body, onLoad) {
    var r = new XMLHttpRequest();
    r.open("POST", url);
    r.withCredentials = false;

    headers = headers || {};
    for (var header in headers) {
      if (headers.hasOwnProperty(header)) {
        r.setRequestHeader(header, headers[header]);
      }
    }

    r.onreadystatechange = function() {
      if (r.readyState === XMLHttpRequest.DONE) {
        onLoad && onLoad(r.status)
      }
    };

    r.send(body);
  }

})();
