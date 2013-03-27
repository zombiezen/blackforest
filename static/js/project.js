(function($) {
    function createBanner(message) {
        var newAlert = $('<div class="alert"></div>');
        newAlert
            .append('<a class="close" data-dismiss="alert" href="#">&times;</a>')
            .append(document.createTextNode(message));
        return newAlert;
    }

    function createAlert(title, message) {
        var newAlert = $('<div class="alert alert-error"></div>');
        newAlert
            .append('<a class="close" data-dismiss="alert" href="#">&times;</a>')
            .append($('<strong></strong>').text(title));
        if (message) {
            newAlert
                .append(" ")
                .append(document.createTextNode(message));
        }
        return newAlert;
    };

    function serializeExclude(form, excludeList) {
        var data = $(form).serializeArray();
        for (var i = 0; i < data.length; ) {
            var removed = false;
            for (var j = 0; j < excludeList.length; j++) {
                if (data[i].name == excludeList[j]) {
                    removed = true;
                    data = data.splice(i, 1);
                    break;
                }
            }
            if (!removed) {
                i++;
            }
        }
        return $.param(data);
    };

    var vcsInput = $('#createform *[name="vcs"], #editform *[name="vcs"]');
    var vcsurlInput = $('#createform input[name="vcsurl"], #editform input[name="vcsurl"]');
    if (vcsInput.val() == "") {
        vcsurlInput.attr("disabled", "disabled");
    }
    vcsInput.change(function(e) {
        if (vcsInput.val() == "") {
            vcsurlInput.attr("disabled", "disabled");
        } else {
            vcsurlInput.removeAttr("disabled");
        }
    });

    $("#createform").submit(function(e) {
        e.preventDefault();
        e.stopPropagation();

        var form = $("#createform");
        var action = form.attr("action");
        var excluded = [];
        if (vcsInput.val() == "") {
            excluded.push("vcsurl");
        }
        $.ajax(action, {
            "data": serializeExclude(form, excluded),
            "type": "POST",
            "success": function(data, status, xhr) {
                var loc = xhr.getResponseHeader("Location");
                window.location = loc + "?shownewbanner=1";
            },
            "error": function(xhr, status, error) {
                $('input[type="submit"]', form).before(createAlert("Server Error").fadeIn());
            }
        });
    });

    $("#editform").submit(function(e) {
        e.preventDefault();
        e.stopPropagation();

        var form = $("#editform");
        var action = form.attr("action");
        var excluded = [];
        if (vcsInput.val() == "") {
            excluded.push("vcsurl");
        }
        $.ajax(action, {
            "data": serializeExclude(form, excluded),
            "type": "PUT",
            "success": function(data, status, xhr) {
                var shortName = $('input[name="shortname"]', form).val();
                window.location = shortName + "?showupdatedbanner=1";
            },
            "error": function(xhr, status, error) {
                $('input[type="submit"]', form).before(createAlert("Server Error").fadeIn());
            }
        });
    });

    // TODO(light): should probably actually parse the query string
    if (window.location.search.indexOf("shownewbanner") != -1) {
        $("body > .container > h1").after(createBanner("Project created").fadeIn());
    } else if (window.location.search.indexOf("showupdatedbanner") != -1) {
        $("body > .container > h1").after(createBanner("Project updated").fadeIn());
    }
})($);
