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

    $("#createform").submit(function(e) {
        e.preventDefault();
        e.stopPropagation();

        var form = $("#createform");
        var action = form.attr("action");
        $.ajax(action, {
            "data": form.serialize(),
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
        $.ajax(action, {
            "data": form.serialize(),
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
