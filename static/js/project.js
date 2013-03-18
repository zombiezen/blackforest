(function($) {
    $("#createform").submit(function(e) {
        e.preventDefault();
        e.stopPropagation();

        var form = $("#createform");
        var action = form.attr("action");
        $.ajax(action, {
            "data": form.serialize(),
            "type": "PUT",
            "success": function() {
                // TODO(light): redirect user to project page
            },
        });
    });
})($);
